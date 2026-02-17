package modeltest

import (
	"testing"

	entity "magento.GO/model/entity"
	productEntity "magento.GO/model/entity/product"
	categoryEntity "magento.GO/model/entity/category"
	salesEntity "magento.GO/model/entity/sales"
)

func TestProduct_TableName(t *testing.T) {
	p := productEntity.Product{}
	if got := p.TableName(); got != "catalog_product_entity" {
		t.Errorf("Product.TableName() = %q, want catalog_product_entity", got)
	}
}

func TestCategory_TableName(t *testing.T) {
	c := categoryEntity.Category{}
	if got := c.TableName(); got != "catalog_category_entity" {
		t.Errorf("Category.TableName() = %q, want catalog_category_entity", got)
	}
}

func TestSalesOrderGrid_TableName(t *testing.T) {
	o := salesEntity.SalesOrderGrid{}
	if got := o.TableName(); got != "sales_order_grid" {
		t.Errorf("SalesOrderGrid.TableName() = %q, want sales_order_grid", got)
	}
}

func TestFlag_TableName(t *testing.T) {
	f := entity.Flag{}
	if got := f.TableName(); got != "flag" {
		t.Errorf("Flag.TableName() = %q, want flag", got)
	}
}

func TestEavAttribute_TableName(t *testing.T) {
	a := entity.EavAttribute{}
	if got := a.TableName(); got != "eav_attribute" {
		t.Errorf("EavAttribute.TableName() = %q, want eav_attribute", got)
	}
}
