package main

type Product struct {
	ID                   int       `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	Category             string    `json:"category"`
	Price                float64   `json:"price"`
	DiscountPercentage   float64   `json:"discountPercentage"`
	Rating               float64   `json:"rating"`
	Stock                int       `json:"stock"`
	Tags                 []string  `json:"tags"`
	Brand                string    `json:"brand"`
	Sku                  string    `json:"sku"`
	Weight               int       `json:"weight"`
	Dimensions           Dimension `json:"dimensions"`
	WarrantyInformation  string    `json:"warrantyInformation"`
	ShippingInformation  string    `json:"shippingInformation"`
	AvailabilityStatus   string    `json:"availabilityStatus"`
	Reviews              []Review  `json:"reviews"`
	ReturnPolicy         string    `json:"returnPolicy"`
	MinimumOrderQuantity int       `json:"minimumOrderQuantity"`
	Meta                 Meta      `json:"meta"`
	Images               []string  `json:"images"`
	Thumbnail            string    `json:"thumbnail"`
}

type Dimension struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Depth  float64 `json:"depth"`
}

type Review struct {
	Rating        int    `json:"rating"`
	Comment       string `json:"comment"`
	Date          string `json:"date"`
	ReviewerName  string `json:"reviewerName"`
	ReviewerEmail string `json:"reviewerEmail"`
}

type Meta struct {
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Barcode   string `json:"barcode"`
	QRCode    string `json:"qrCode"`
}

type apiResponse struct {
	Products []Product `json:"products"`
}
