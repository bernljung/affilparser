package main

type tradedoubler struct {
	ProductHeader struct {
		TotalHits int
	}
	Products []struct {
		Name         string
		ProductImage struct {
			URL string
		}
		Language    string
		Description string
		Brand       string
		Identifiers struct {
			EAN string
			MPN string
			SKU string
		}
		GroupingID string
		Fields     []struct {
			Name  string
			Value string
		}
		Offers []struct {
			FeedID       int
			ProductURL   string
			PriceHistory []struct {
				Price struct {
					Value    string
					Currency string
				}
				Date int
			}
			Modified        int
			InStock         int
			Availability    string
			ShippingCost    string
			SourceProductID string
			ProgramLogo     string
			ProgramName     string
			ID              string
		}
		Categories []struct {
			Name           string
			TDCategoryName string
			ID             int
		}
	}
}

func (n tradedoubler) parseProducts(f *feed) ([]product, error) {
	var err error
	// var jsonData map[string]interface{}

	// err = json.Unmarshal(f.FeedData, &jsonData)
	// if err != nil {
	// 	return err
	// }
	// products, ok := jsonData[f.Network.getProductsField()].([]interface{})
	return make([]product, 0), err
}
