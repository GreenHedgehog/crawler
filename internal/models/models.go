package models

import "crawler/rpc/crawler"

type (
	Product      = crawler.Product
	SearchParams = crawler.SearchParams
)

type SearchField = crawler.SearchParams_Field

const (
	SearchByDefault     SearchField = crawler.SearchParams_DEFAULT // default database order
	SearchByPrice       SearchField = crawler.SearchParams_PRICE
	SearchByPriceChange SearchField = crawler.SearchParams_PRICE_CHANGES_COUNTER
	SearchByLastUpdate  SearchField = crawler.SearchParams_LAST_UPDATE_TS
	SearchByName        SearchField = crawler.SearchParams_NAME
)

type SearchOrder = crawler.SearchParams_Order

const (
	SearchAscending  SearchOrder = crawler.SearchParams_ASC
	SearchDescending SearchOrder = crawler.SearchParams_DESC
)
