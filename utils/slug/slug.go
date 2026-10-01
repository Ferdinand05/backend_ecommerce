package slug

import (
	"regexp"
	"strings"
)

func MakeSlug(s string) string {
	// 1. Ubah ke huruf kecil semua
	slug := strings.ToLower(s)

	// 2. Ganti karakter khusus / simbol tertentu menjadi spasi atau langsung dihapus
	// Mengganti tanda '&' menjadi 'and' (opsional, tergantung kebutuhan)
	slug = strings.ReplaceAll(slug, "&", "and")

	// 3. Hapus semua karakter yang BUKAN huruf (a-z), angka (0-9), atau spasi/tanda hubung
	reg := regexp.MustCompile(`[^a-z0-9\s-_]`)
	slug = reg.ReplaceAllString(slug, "")

	// 4. Ganti spasi, underscore (_), atau tanda hubung beruntun menjadi satu tanda hubung (-)
	regSpace := regexp.MustCompile(`[\s-_]+`)
	slug = regSpace.ReplaceAllString(slug, "-")

	// 5. Bersihkan tanda hubung di awal atau di akhir string jika ada
	slug = strings.Trim(slug, "-")

	return slug
}