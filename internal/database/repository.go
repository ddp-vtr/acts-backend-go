package database

import (
	"acts-backend-go/internal/model"
	"strings"
	"sync"
)

// In-Memory Catalog Repository với Filter chuẩn hóa (Interface sẵn sàng cắm sang SQLite / PostgreSQL)
type Repository struct {
	mu       sync.RWMutex
	products []model.Product
}

func NewRepository() *Repository {
	repo := &Repository{
		products: make([]model.Product, 0),
	}
	repo.seedCatalog()
	return repo
}

// Lấy danh sách sản phẩm nổi bật mặc định khi khách hỏi mở
func (r *Repository) GetFeaturedProducts(limit int) []model.Product {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.products) {
		limit = 3
	}
	featured := make([]model.Product, limit)
	copy(featured, r.products[:limit])
	return featured
}

// Lọc sản phẩm chuẩn SQL logic: lọc cứng Category, Price, Stock > 0 và kết hợp Semantic Score
func (r *Repository) SearchProducts(category string, maxPrice int, undertone string, query string, limit int) []model.Product {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []model.Product
	qLower := strings.ToLower(query)

	for _, p := range r.products {
		// 1. Hard Filter: Phải còn tồn kho
		if p.StockQuantity <= 0 {
			continue
		}

		// 2. Hard Filter: Category (nếu có yêu cầu)
		if category != "" && !strings.EqualFold(p.Category, category) {
			continue
		}

		// 3. Hard Filter: MaxPrice (nếu có yêu cầu)
		if maxPrice > 0 && p.Price > maxPrice {
			continue
		}

		// 4. Hard Filter: Undertone (nếu có yêu cầu)
		if undertone != "" && p.Undertone != "" && !strings.EqualFold(p.Undertone, undertone) {
			continue
		}

		// 5. Keyword / Semantic matching
		matched := false
		if qLower == "" {
			matched = true
		} else {
			if strings.Contains(strings.ToLower(p.Name), qLower) ||
				strings.Contains(strings.ToLower(p.Desc), qLower) ||
				strings.Contains(strings.ToLower(p.Brand), qLower) {
				matched = true
			} else {
				for _, kw := range p.Keywords {
					if strings.Contains(qLower, strings.ToLower(kw)) {
						matched = true
						break
					}
				}
			}
		}

		if matched {
			results = append(results, p)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results
}

func (r *Repository) seedCatalog() {
	r.products = []model.Product{
		{
			ID: 9, SkyID: 780398, Category: "son", Brand: "BBIA",
			Name: "Bbia Last Velvet Lip Tint", Desc: "Chất son kem lì mịn như nhung, 5 tone màu quốc dân tôn da châu Á, bám màu 12h.",
			Price: 160000, PriceDisplay: "160.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-VELVET-TINT-RENEWAL-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "last", "velvet", "lip", "tint", "#25", "#24", "#12"},
		},
		{
			ID: 1, SkyID: 769345, Category: "mat", Brand: "BBIA",
			Name: "BBIA Nudy Slim Eyeliner", Desc: "Đầu chì siêu mảnh 1.5mm chống lem trôi, dễ kẻ mí trong và khóe mắt sắc nét suốt 24h.",
			Price: 189000, PriceDisplay: "189.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2022/04/SKU-WEB_BBIA-NUDY-SLIM-EYELINER.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "nudy", "slim", "eyeliner"},
		},
		{
			ID: 2, SkyID: 775177, Category: "phan", Brand: "BBIA",
			Name: "BBIA Chaltteok Blush", Desc: "Phấn má hồng dạng phấn dẻo mềm mịn, dễ tán tiệp màu tự nhiên chuẩn phong cách Hàn Quốc.",
			Price: 235000, PriceDisplay: "235.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2023/02/BBIA-CHALTTEOK-BLUSH.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "chaltteok", "blush", "01", "02", "03", "04"},
		},
		{
			ID: 3, SkyID: 769346, Category: "son", Brand: "BBIA",
			Name: "BBIA Lip Oil Essence", Desc: "Tinh chất dưỡng môi cấp ẩm chuyên sâu, phục hồi môi khô nứt nẻ, căng bóng tự nhiên.",
			Price: 199000, PriceDisplay: "199.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2022/04/SKU-WEB_BBIA-LIP-OIL-ESSENCE-1.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "lip", "oil", "essence"},
		},
		{
			ID: 4, SkyID: 773363, Category: "phan", Brand: "BBIA",
			Name: "BBIA Last Contour Pen Eyeliner", Desc: "Bút dạ kẻ bọng mắt sắc nét tự nhiên, tạo chiều sâu cho đôi mắt to tròn long lanh.",
			Price: 165000, PriceDisplay: "165.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2022/09/Sky007-SKU__-Contour-Pen-Eyeliner.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "last", "contour", "pen", "eyeliner"},
		},
		{
			ID: 5, SkyID: 785696, Category: "phan", Brand: "BBIA",
			Name: "BBIA Eau Powder Pact", Desc: "Phấn phủ hiệu chỉnh sắc da, làm mờ lỗ chân lông và kiềm dầu kháng nước suốt 12h.",
			Price: 248000, PriceDisplay: "248.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2025/10/SKU_BBIA-EAU-POWDER-PACT-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "eau", "powder", "pact"},
		},
		{
			ID: 6, SkyID: 780277, Category: "phan", Brand: "BBIA",
			Name: "BBIA Eau Glow Cushion", Desc: "Phấn nước che khuyết điểm hoàn hảo, căng bóng rạng rỡ chuẩn phong cách Glow Hàn Quốc.",
			Price: 495000, PriceDisplay: "495.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-EAU-GLOW-CUSHION.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "eau", "glow", "cushion"},
		},
		{
			ID: 7, SkyID: 781994, Category: "phan", Brand: "BBIA",
			Name: "BBIA Eau Glow Cushion Mini", Desc: "Cushion mini tiện lợi mang theo, lớp nền mỏng nhẹ tự nhiên dặm lại mọi lúc mọi nơi.",
			Price: 230000, PriceDisplay: "230.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2025/04/SKU_BBIA-EAU-GLOW-CUSHION-Mini.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "eau", "glow", "cushion", "mini"},
		},
		{
			ID: 8, SkyID: 780285, Category: "mat", Brand: "BBIA",
			Name: "BBIA Essential Eye Palette", Desc: "Bảng phấn mắt đa năng tone màu thời thượng, chất phấn nhung mịn bám màu cả ngày.",
			Price: 488000, PriceDisplay: "488.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-ESSENTIAL-EYE-PALETTE.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "essential", "eye", "palette"},
		},
		{
			ID: 10, SkyID: 780404, Category: "son", Brand: "BBIA",
			Name: "BBIA Glow Tint", Desc: "Son tint bóng căng mọng tự nhiên, không gây dính môi, giữ màu rạng rỡ tươi tắn.",
			Price: 189000, PriceDisplay: "189.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-GLOW-TINT-RENEWAL.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "glow", "tint"},
		},
		{
			ID: 11, SkyID: 780238, Category: "son", Brand: "BBIA",
			Name: "BBIA L'eau Tint", Desc: "Son tint nước mỏng nhẹ như sương, công nghệ phản chiếu ánh sáng cho môi căng mọng.",
			Price: 209000, PriceDisplay: "209.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LEAU-TINT-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "l'eau", "tint"},
		},
		{
			ID: 12, SkyID: 780369, Category: "mat", Brand: "BBIA",
			Name: "BBIA Last Auto Eyebrow Pencil", Desc: "Chì kẻ mày đầu tam giác định hình chân mày tự nhiên, chống trôi lem mồ hôi.",
			Price: 66000, PriceDisplay: "66.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-AUTO-EYEBROW-PENCIL-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "last", "auto", "eyebrow", "pencil"},
		},
		{
			ID: 13, SkyID: 780298, Category: "mat", Brand: "BBIA",
			Name: "BBIA Last Auto Gel Eyeliner", Desc: "Chì kẻ mắt dạng gel sắc nét, công nghệ khóa màu chống lem trôi nước suốt ngày dài.",
			Price: 121000, PriceDisplay: "121.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-AUTO-GEL-EYELINER-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "last", "auto", "gel", "eyeliner"},
		},
		{
			ID: 14, SkyID: 780316, Category: "mat", Brand: "BBIA",
			Name: "BBIA Last Auto Gel Eyeliner Slim", Desc: "Chì kẻ mắt dạng gel đầu siêu mảnh 2mm kẻ viền mắt trong và đường liner chuẩn xác.",
			Price: 154000, PriceDisplay: "154.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-AUTO-GEL-EYELINER-SLIM.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "last", "auto", "gel", "eyeliner", "slim"},
		},
		{
			ID: 15, SkyID: 782798, Category: "phan", Brand: "BBIA",
			Name: "BBIA Last Blush", Desc: "Phấn má hồng kết cấu siêu mịn nhẹ tênh, tạo hiệu ứng má ửng hồng trong trẻo.",
			Price: 160000, PriceDisplay: "160.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2025/06/SKU_BBIA-LAST-BLUSH-RENEWAL.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "last", "blush"},
		},
		{
			ID: 16, SkyID: 786660, Category: "son", Brand: "BBIA",
			Name: "BBIA Lucky Shine Tint 4.5g", Desc: "Son tint bóng phiên bản giới hạn, nhũ mịn lấp lánh phản chiếu ánh sáng sang trọng.",
			Price: 209000, PriceDisplay: "209.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2026/03/WEB_BBIA-LUCKY-SHINE-TINT-01-08.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "lucky", "shine", "tint", "4.5g"},
		},
		{
			ID: 17, SkyID: 781002, Category: "son", Brand: "BBIA",
			Name: "BBIA Air Fit Tint – MUKBANG Edition", Desc: "Son kem lì phiên bản đặc biệt Mukbang, kháng dầu mỡ, ăn uống không lo trôi son.",
			Price: 165000, PriceDisplay: "165.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/11/SKU_MUKBANG_BBIA-AIR-FIT-TINT.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "air", "fit", "tint", "mukbang", "edition"},
		},
		{
			ID: 18, SkyID: 780391, Category: "phan", Brand: "BBIA",
			Name: "BBIA Last Contour Stick", Desc: "Thỏi tạo khối 2 đầu tiện lợi định hình sống mũi và gương mặt V-line thon gọn.",
			Price: 220000, PriceDisplay: "220.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-CONTOUR-STICK-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "last", "contour", "stick"},
		},
		{
			ID: 19, SkyID: 782037, Category: "mat", Brand: "BBIA",
			Name: "BBIA Last Eye Palette", Desc: "Bảng mắt 8 ô màu đa hiệu ứng lì nhũ, phối màu mắt phong cách Clean Girl chuẩn Hàn.",
			Price: 390000, PriceDisplay: "390.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2025/04/SKU_BBIA-LAST-EYE-PALETTE-2.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "last", "eye", "palette"},
		},
		{
			ID: 20, SkyID: 780354, Category: "mat", Brand: "BBIA",
			Name: "BBIA Last Pen Eyeliner", Desc: "Bút dạ kẻ mắt nét mảnh mực đều không đứt nét, khô nhanh trong 3 giây chống nước.",
			Price: 120000, PriceDisplay: "120.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/BBIA-SKU_Last-Pen-Eyeliner.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "last", "pen", "eyeliner"},
		},
		{
			ID: 21, SkyID: 780342, Category: "mat", Brand: "BBIA",
			Name: "BBIA Last Powder Pencil", Desc: "Chì kẻ mày dạng bột nén tạo sợi lông mày tự nhiên như điêu khắc chân thật.",
			Price: 130000, PriceDisplay: "130.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-LAST-POWDER-PENCIL.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "last", "powder", "pencil"},
		},
		{
			ID: 22, SkyID: 780346, Category: "mat", Brand: "BBIA",
			Name: "BBIA Never Die Brush Eyeliner", Desc: "Bút kẻ mắt nước kháng nước Never Die đỉnh cao, nét sắc bén bám chặt 24h không lem.",
			Price: 165000, PriceDisplay: "165.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-NEVER-DIE-BRUSH-LINER.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "never", "die", "brush", "eyeliner"},
		},
		{
			ID: 23, SkyID: 780349, Category: "mat", Brand: "BBIA",
			Name: "BBIA Never Die Mascara", Desc: "Mascara làm cong dày và dài mi chuyên nghiệp, công thức chống thấm nước đỉnh cao.",
			Price: 198000, PriceDisplay: "198.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-NEVER-DIE-MASCARA-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "never", "die", "mascara"},
		},
		{
			ID: 24, SkyID: 781932, Category: "mat", Brand: "BBIA",
			Name: "BBIA Never Die Mascara Extreme", Desc: "Mascara phiên bản Extreme siêu làm dài và tơi mi, không vón cục nặng mắt.",
			Price: 192000, PriceDisplay: "192.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2025/04/SKU_BBIA-NEVER-DIE-MASCARA-EXTREME.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "never", "die", "mascara", "extreme"},
		},
		{
			ID: 25, SkyID: 771153, Category: "mat", Brand: "BBIA",
			Name: "BBIA Never Die Mascara Slim", Desc: "Mascara đầu chuốt siêu mảnh len lỏi chải tơi từng sợi mi dưới và khóe mi sắc nét.",
			Price: 139000, PriceDisplay: "139.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2022/06/SKU_BBIA-NEVER-DIE-MASCARA-SLIM-1.png",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "never", "die", "mascara", "slim"},
		},
		{
			ID: 26, SkyID: 780654, Category: "son", Brand: "BBIA",
			Name: "BBIA Over Glaze", Desc: "Son bóng tráng gương Over Glaze tạo hiệu ứng môi thủy tinh mọng nước quyến rũ.",
			Price: 196000, PriceDisplay: "196.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/10/SHORT-VER_BBIA-Over-Glaze_CARAMEL_01.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "over", "glaze"},
		},
		{
			ID: 27, SkyID: 780246, Category: "son", Brand: "BBIA",
			Name: "BBIA Over Glaze Stick", Desc: "Son thỏi thạch tan chảy dưỡng ẩm làm đầy rãnh môi, lên màu trong veo ngọt ngào.",
			Price: 149000, PriceDisplay: "149.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/BBIA-Over-glaze-stick.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "over", "glaze", "stick"},
		},
		{
			ID: 28, SkyID: 780331, Category: "phan", Brand: "BBIA",
			Name: "BBIA Ready To Wear Downy Cheek", Desc: "Má hồng kem dạng thỏi tan mịn vào da, tạo gò má ửng hồng tự nhiên bóng khỏe.",
			Price: 154000, PriceDisplay: "154.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU-WEB_RTW-DOWNY-CHEEK.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "ready", "wear", "downy", "cheek"},
		},
		{
			ID: 29, SkyID: 780289, Category: "mat", Brand: "BBIA",
			Name: "BBIA Ready To Wear Eye Palette", Desc: "Bảng phấn mắt phối sẵn tông ấm tông lạnh thời thượng, dễ dùng cho mọi layout.",
			Price: 319000, PriceDisplay: "319.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU-WEB_BBIA-RTW-EYE-PALETTE-11-APPLE-EDITION.jpg",
			Undertone: "neutral", Keywords: []string{"bbia", "mat", "ready", "wear", "eye", "palette"},
		},
		{
			ID: 30, SkyID: 780256, Category: "son", Brand: "BBIA",
			Name: "BBIA Ready To Wear Water Lipstick", Desc: "Son thỏi dưỡng ẩm có màu mỏng nhẹ mọng nước, cho đôi môi vẻ tươi tắn suốt ngày.",
			Price: 170000, PriceDisplay: "170.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-RTW-WATER-LIPSTICK.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "ready", "wear", "water", "lipstick"},
		},
		{
			ID: 31, SkyID: 781037, Category: "son", Brand: "BBIA",
			Name: "BBIA Water Fit Tint – MUKBANG Edition", Desc: "Son tint nước bền màu phiên bản Mukbang, kháng nước kháng lem tuyệt đối.",
			Price: 165000, PriceDisplay: "165.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/11/SKU_MUKBANG_BBIA-WATER-FIT-TINT.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "water", "fit", "tint", "mukbang", "edition"},
		},
		{
			ID: 32, SkyID: 780281, Category: "phan", Brand: "BBIA",
			Name: "BBIA Eau Stay Concealer – Eau Edition", Desc: "Kem che khuyết điểm kiêm dưỡng da Eau Edition, che phủ quầng thâm và mụn đỏ.",
			Price: 239000, PriceDisplay: "239.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2024/09/SKU_BBIA-EAU-STAY-CONCEALER.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "eau", "stay", "concealer", "edition"},
		},
		{
			ID: 33, SkyID: 551001, Category: "son", Brand: "BBIA",
			Name: "BBIA Lip Oil Balm #01 Shea Butter", Desc: "Son dưỡng bơ hạt mỡ phục hồi môi nứt nẻ, làm mềm môi tức thì sau 1 đêm.",
			Price: 165000, PriceDisplay: "165.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2021/04/Bbia-Lip-Oil-Balm.png",
			Undertone: "neutral", Keywords: []string{"bbia", "son", "lip", "oil", "balm", "#01", "shea", "butter"},
		},
		{
			ID: 34, SkyID: 547975, Category: "phan", Brand: "BBIA",
			Name: "BBIA Eau Matte Cushion 15g", Desc: "Phấn nước kiềm dầu mờ lì Eau Matte, giữ lớp nền khô thoáng không bí da cả ngày.",
			Price: 488000, PriceDisplay: "488.000₫", StockQuantity: 50,
			ImageUrl:  "https://www.sky007.vn/wp-content/uploads/2021/04/SKU_BBIA-EAU-MATTE-CUSHION-2.png",
			Undertone: "neutral", Keywords: []string{"bbia", "phan", "eau", "matte", "cushion", "15g"},
		},
	}
}
