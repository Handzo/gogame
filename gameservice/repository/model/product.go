package model

import (
	"fmt"

	basemodel "github.com/Handzo/gogame/common/model"
	"github.com/go-pg/pg/v9"
)

type Product struct {
	basemodel.BaseModel
	Title       string `pg:",notnull"`
	Description string
	Price       uint32   `pg:",notnull,default:100"`
	Currency    Currency `pg:",notnull,type:currency"`
	Goods       []*Good  `pg:"many2many:product_to_goods"`
}

func (Product) Prepare(*pg.DB, bool) error {
	return nil
}

func (Product) Sync(*pg.DB, bool) error {
	return nil
}

func (Product) Populate(db *pg.DB, force bool) error {
	goodItems := []*GoodItem{
		&GoodItem{Title: "Nuts", Description: "Nuts for playing belka"},
		&GoodItem{Title: "Gold", Description: "Gold for playing belka"},
		&GoodItem{Title: "VIP", Description: "VIP for playing belka"},
	}

	for _, g := range goodItems {
		if err := db.Insert(g); err != nil {
			return err
		}
		fmt.Println(g.Id)
	}

	goods := []*Good{
		&Good{GoodItemId: goodItems[0].Id, Amount: 10},
		&Good{GoodItemId: goodItems[0].Id, Amount: 100},
		&Good{GoodItemId: goodItems[0].Id, Amount: 1000},
		&Good{GoodItemId: goodItems[1].Id, Amount: 100},
		&Good{GoodItemId: goodItems[1].Id, Amount: 1000},
		&Good{GoodItemId: goodItems[1].Id, Amount: 10000},
		&Good{GoodItemId: goodItems[2].Id, Amount: 1},
	}

	for _, g := range goods {
		if err := db.Insert(g); err != nil {
			return err
		}
		fmt.Println(g.Id)
	}

	products := []*Product{
		&Product{Title: "Nuts", Description: "Small pack of nuts", Price: 99, Currency: GOLD},
		&Product{Title: "Nuts", Description: "Medium pack of nuts", Price: 990, Currency: GOLD},
		&Product{Title: "Nuts", Description: "Big pack of nuts", Price: 9900, Currency: GOLD},
		&Product{Title: "Gold", Description: "Small pack of nuts", Price: 99, Currency: USD},
		&Product{Title: "Gold", Description: "Medium pack of nuts", Price: 990, Currency: USD},
		&Product{Title: "Gold", Description: "Big pack of nuts", Price: 9900, Currency: USD},
		&Product{Title: "VIP", Description: "Small pack of nuts", Price: 9900, Currency: GOLD},
	}

	for _, p := range products {
		if err := db.Insert(p); err != nil {
			return err
		}
		fmt.Println(p.Id)
	}

	productsGood := []*ProductToGood{
		&ProductToGood{ProductId: products[0].Id, GoodId: goods[0].Id},
		&ProductToGood{ProductId: products[1].Id, GoodId: goods[0].Id},
		&ProductToGood{ProductId: products[2].Id, GoodId: goods[0].Id},
		&ProductToGood{ProductId: products[3].Id, GoodId: goods[1].Id},
		&ProductToGood{ProductId: products[4].Id, GoodId: goods[1].Id},
		&ProductToGood{ProductId: products[5].Id, GoodId: goods[1].Id},
		&ProductToGood{ProductId: products[6].Id, GoodId: goods[2].Id},
	}

	for _, p := range productsGood {
		if err := db.Insert(p); err != nil {
			return err
		}
	}

	return nil
}
