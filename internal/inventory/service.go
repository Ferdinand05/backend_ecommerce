package inventory

import (
	"context"
	"ferdinand/ecommerce/database"
	"ferdinand/ecommerce/internal/models"
	productvariant "ferdinand/ecommerce/internal/product_variant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service interface {
	GetInventory(
		ctx context.Context,
		productID uuid.UUID,
		variantID uuid.UUID,
	) (InventoryResponse, error)

	ListMovements(
		ctx context.Context,
		productID uuid.UUID,
		variantID uuid.UUID,
	) ([]StockMovementResponse, error)

	GetMovement(
		ctx context.Context,
		productID uuid.UUID,
		variantID uuid.UUID,
		movementID uuid.UUID,
	) (StockMovementResponse, error)

	CreateMovement(
		ctx context.Context,
		productID uuid.UUID,
		variantID uuid.UUID,
		req CreateStockMovementRequest,
	) (StockMovementResponse, error)
}

type service struct {
	itemRepo     InventoryRepository
	movementRepo StockMovementRepository
	variantRepo  productvariant.Repository
	db           *gorm.DB
}

func NewService(
	itemRepo InventoryRepository,
	movementRepo StockMovementRepository,
	variantRepo productvariant.Repository,
	db *gorm.DB,
) *service {
	return &service{
		itemRepo:     itemRepo,
		movementRepo: movementRepo,
		variantRepo:  variantRepo,
		db:           db,
	}
}

func (s *service) ownedVariant(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) (models.ProductVariant, error) {

	variant, err := s.variantRepo.FindByID(ctx, variantID)
	if err != nil {
		return models.ProductVariant{}, err
	}

	if variant.ProductID != productID {
		return models.ProductVariant{}, productvariant.ErrorProductVariantNotFound
	}

	return variant, nil
}

func deltaFromType(movementType StockMovementType) (int, bool) {

	switch movementType {
	case StockMovementRestock, StockMovementReturn, StockMovementOrderCancel, StockMovementAdjustment:
		return 1, true
	case StockMovementSale, StockMovementDamage:
		return -1, true
	default:
		return 0, false
	}
}

func toInventoryResponse(item models.InventoryItem) InventoryResponse {
	return InventoryResponse{
		ID:               item.ID,
		ProductVariantID: item.ProductVariantID,
		Quantity:         item.Quantity,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func toStockMovementResponse(movement models.StockMovement) StockMovementResponse {
	return StockMovementResponse{
		ID:              movement.ID,
		InventoryItemID: movement.InventoryItemID,
		Type:            movement.Type,
		Quantity:        movement.Quantity,
		Note:            movement.Note,
		ReferenceType:   movement.ReferenceType,
		ReferenceID:     movement.ReferenceID,
		CreatedAt:       movement.CreatedAt,
	}
}

func toStockMovementResponses(movements []models.StockMovement) []StockMovementResponse {
	responses := make([]StockMovementResponse, len(movements))
	for i, m := range movements {
		responses[i] = toStockMovementResponse(m)
	}

	return responses
}

func (s *service) GetInventory(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) (InventoryResponse, error) {

	if _, err := s.ownedVariant(ctx, productID, variantID); err != nil {
		return InventoryResponse{}, err
	}

	item, err := s.itemRepo.FindByVariantID(ctx, variantID)
	if err != nil {
		return InventoryResponse{}, err
	}

	return toInventoryResponse(item), nil
}

func (s *service) ListMovements(ctx context.Context, productID uuid.UUID, variantID uuid.UUID) ([]StockMovementResponse, error) {

	if _, err := s.ownedVariant(ctx, productID, variantID); err != nil {
		return nil, err
	}

	item, err := s.itemRepo.FindByVariantID(ctx, variantID)
	if err != nil {
		return nil, err
	}

	movements, err := s.movementRepo.FindAllByInventoryID(ctx, item.ID)
	if err != nil {
		return nil, err
	}

	return toStockMovementResponses(movements), nil
}

func (s *service) GetMovement(ctx context.Context, productID uuid.UUID, variantID uuid.UUID, movementID uuid.UUID) (StockMovementResponse, error) {

	if _, err := s.ownedVariant(ctx, productID, variantID); err != nil {
		return StockMovementResponse{}, err
	}

	item, err := s.itemRepo.FindByVariantID(ctx, variantID)
	if err != nil {
		return StockMovementResponse{}, err
	}

	movement, err := s.movementRepo.FindByID(ctx, movementID)
	if err != nil {
		return StockMovementResponse{}, err
	}

	if movement.InventoryItemID != item.ID {
		return StockMovementResponse{}, ErrorStockMovementNotFound
	}

	return toStockMovementResponse(movement), nil
}

func (s *service) CreateMovement(ctx context.Context, productID uuid.UUID, variantID uuid.UUID, req CreateStockMovementRequest) (StockMovementResponse, error) {

	if _, err := s.ownedVariant(ctx, productID, variantID); err != nil {
		return StockMovementResponse{}, err
	}

	if req.Quantity <= 0 {
		return StockMovementResponse{}, ErrorInvalidMovementQuantity
	}

	delta, ok := deltaFromType(req.Type)
	if !ok {
		return StockMovementResponse{}, ErrorInvalidMovementType
	}

	amount := req.Quantity * delta

	var created models.StockMovement

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txCtx := database.InjectTx(ctx, tx)

		item, err := s.itemRepo.FindByVariantIDForUpdate(txCtx, variantID)
		if err != nil {
			return err
		}

		newQuantity := item.Quantity + amount
		if newQuantity < 0 {
			return ErrorInsufficientStock
		}

		if err := s.itemRepo.UpdateQuantity(txCtx, item.ID, newQuantity); err != nil {
			return err
		}

		created = models.StockMovement{
			ID:              uuid.New(),
			InventoryItemID: item.ID,
			Type:            string(req.Type),
			Quantity:        amount,
			Note:            req.Note,
			ReferenceType:   req.ReferenceType,
			ReferenceID:     req.ReferenceID,
		}

		return s.movementRepo.Create(txCtx, created)
	})

	if err != nil {
		return StockMovementResponse{}, err
	}

	return toStockMovementResponse(created), nil
}
