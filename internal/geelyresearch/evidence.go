package geelyresearch

// EvidenceConfidence describes confidence in the bounded claim made by an
// evidence entry, not confidence in any vehicle's compatibility.
type EvidenceConfidence string

const EvidenceHigh EvidenceConfidence = "high"

type Evidence struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Publisher   string             `json:"publisher"`
	URL         string             `json:"url"`
	PublishedAt string             `json:"published_at,omitempty"`
	RetrievedAt string             `json:"retrieved_at"`
	Scope       string             `json:"scope"`
	Confidence  EvidenceConfidence `json:"confidence"`
}

var publicEvidence = []Evidence{
	{
		ID:          "geely-xingyue-l-launch-2021",
		Title:       "Geely Launches Newest High-End Flagship SUV – Xingyue L",
		Publisher:   "Geely Auto",
		URL:         "https://global.geely.com/en/news/2021/geely-launches-suv-xingyuel",
		PublishedAt: "2021-07-20",
		RetrievedAt: "2026-11-01",
		Scope:       "В описании запуска указаны Milky Way OS и многоэкранный кокпит Xingyue L. Источник не задаёт сборки ПО или аппаратные ревизии 2022–2024 годов.",
		Confidence:  EvidenceHigh,
	},
	{
		ID:        "geely-xingyue-l-launch-specifications-2021",
		Title:     "Geely Xingyue L launch specifications",
		Publisher: "Geely Auto",
		URL:       "https://dm30webimages.geely.com/GeelyPromotion/KX11/2021/pzb1029.pdf",
		// Publication date is not stated in the PDF body; 2021 is indicated by its official URL path.
		RetrievedAt: "2026-11-01",
		Scope:       "В таблице начальных комплектаций перечислено несколько вариантов и Snapdragon 8155. Она не подтверждает соответствие года, рынка и сборки для 2022–2024 годов.",
		Confidence:  EvidenceHigh,
	},
	{
		ID:          "geely-xingyue-l-specifications-2025",
		Title:       "Geely 2025 Xingyue L Specification Table",
		Publisher:   "Geely Auto",
		URL:         "https://global.geely.com/-/media/project/web-portal/parallel-car/xingyuel-01/geely-xingyue-l-dongfangyao-specification-table.pdf",
		PublishedAt: "2025",
		RetrievedAt: "2026-11-01",
		Scope:       "Для перечисленных вариантов 2025 года указаны Galaxy OS 2.0 и Snapdragon 8155. Это не подтверждение конфигураций 2022–2024 годов.",
		Confidence:  EvidenceHigh,
	},
	{
		ID:          "geely-monjaro-current-product-page",
		Title:       "GEELY MONJARO",
		Publisher:   "Geely Auto",
		URL:         "https://global.geely.com/en/models/monjaro",
		RetrievedAt: "2026-11-01",
		Scope:       "Текущая страница указывает Geely ONE OS 2.0 и Snapdragon 8155 и предупреждает о различиях по рынкам. Она не сопоставляет сборки 2022–2024 годов.",
		Confidence:  EvidenceHigh,
	},
	{
		ID:          "geely-xingyue-l-monjaro-naming-2026",
		Title:       "Geely Auto Exceeds 3.02 Million Vehicle Sales in 2025, Sets New Growth Targets for 2026",
		Publisher:   "Geely Auto",
		URL:         "https://global.geely.com/en/news/2026/geely-auto-sales-2025",
		PublishedAt: "2026",
		RetrievedAt: "2026-11-01",
		Scope:       "Публикация связывает название Xingyue L с Monjaro на зарубежных рынках. Она не доказывает одинаковую мультимедийную систему на разных рынках и годах.",
		Confidence:  EvidenceHigh,
	},
}

// PublicEvidence returns a defensive copy of the source catalog. Each entry
// states the narrow claim supported by the source and its retrieval date.
func PublicEvidence() []Evidence {
	return append([]Evidence(nil), publicEvidence...)
}

func evidenceIDs() map[string]struct{} {
	ids := make(map[string]struct{}, len(publicEvidence))
	for _, source := range publicEvidence {
		ids[source.ID] = struct{}{}
	}
	return ids
}
