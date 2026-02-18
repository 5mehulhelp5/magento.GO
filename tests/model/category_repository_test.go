package modeltest

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	entity "magento.GO/model/entity"
	categoryEntity "magento.GO/model/entity/category"
	categoryRepo "magento.GO/model/repository/category"
)

func categoryRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&entity.EavAttribute{},
		&categoryEntity.Category{},
		&categoryEntity.CategoryProduct{},
		&categoryEntity.CategoryVarchar{},
		&categoryEntity.CategoryInt{},
		&categoryEntity.CategoryText{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedCategoryAttrs(t *testing.T, db *gorm.DB) {
	t.Helper()
	nameLabel := "Name"
	urlLabel := "URL Key"
	attrs := []entity.EavAttribute{
		{AttributeID: 41, EntityTypeID: 3, AttributeCode: "name", BackendType: "varchar", FrontendLabel: &nameLabel},
		{AttributeID: 119, EntityTypeID: 3, AttributeCode: "url_key", BackendType: "varchar", FrontendLabel: &urlLabel},
		{AttributeID: 42, EntityTypeID: 3, AttributeCode: "is_active", BackendType: "int"},
	}
	for _, a := range attrs {
		if err := db.Create(&a).Error; err != nil {
			t.Fatalf("create attr %s: %v", a.AttributeCode, err)
		}
	}
}

func seedCategory(t *testing.T, db *gorm.DB, parentID uint, path string, level int, name string) categoryEntity.Category {
	t.Helper()
	cat := categoryEntity.Category{
		AttributeSetID: 1,
		ParentID:       parentID,
		Path:           path,
		Position:       1,
		Level:          level,
		ChildrenCount:  0,
	}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if name != "" {
		if err := db.Create(&categoryEntity.CategoryVarchar{
			AttributeID: 41, StoreID: 0, EntityID: cat.EntityID, Value: name,
		}).Error; err != nil {
			t.Fatalf("create name varchar: %v", err)
		}
	}
	return cat
}

func TestCategoryRepository_FetchAllWithAttributes(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)

	cats, err := repo.FetchAllWithAttributes(0)
	if err != nil {
		t.Fatalf("FetchAllWithAttributes empty: %v", err)
	}
	if len(cats) != 0 {
		t.Errorf("expected 0 categories, got %d", len(cats))
	}

	seedCategory(t, db, 0, "1", 0, "Root")
	seedCategory(t, db, 1, "1/2", 1, "Electronics")
	repo.InvalidateCache()

	cats, err = repo.FetchAllWithAttributes(0)
	if err != nil {
		t.Fatalf("FetchAllWithAttributes: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}

	foundNames := map[string]bool{}
	for _, c := range cats {
		for _, v := range c.Varchars {
			if v.AttributeID == 41 {
				foundNames[v.Value] = true
			}
		}
	}
	if !foundNames["Root"] || !foundNames["Electronics"] {
		t.Errorf("expected Root and Electronics in varchars, got %v", foundNames)
	}
}

func TestCategoryRepository_FetchAllWithAttributesMap(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)

	cat := seedCategory(t, db, 0, "1", 0, "Mapped Cat")

	catMap, err := repo.FetchAllWithAttributesMap(0)
	if err != nil {
		t.Fatalf("FetchAllWithAttributesMap: %v", err)
	}
	if len(catMap) != 1 {
		t.Fatalf("expected 1, got %d", len(catMap))
	}
	cwa, ok := catMap[cat.EntityID]
	if !ok {
		t.Fatal("category not found in map")
	}
	if nameAttr, ok := cwa.Attributes["name"]; ok {
		if nameAttr["value"] != "Mapped Cat" {
			t.Errorf("name = %v, want Mapped Cat", nameAttr["value"])
		}
	} else {
		t.Error("name attribute missing from flat map")
	}
}

func TestCategoryRepository_GetByIDsWithAttributes(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)

	c1 := seedCategory(t, db, 0, "1", 0, "Cat A")
	c2 := seedCategory(t, db, 0, "1/2", 1, "Cat B")
	_ = seedCategory(t, db, 0, "1/3", 1, "Cat C")

	cats, err := repo.GetByIDsWithAttributes([]uint{c1.EntityID, c2.EntityID}, 0)
	if err != nil {
		t.Fatalf("GetByIDsWithAttributes: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("expected 2, got %d", len(cats))
	}
	ids := map[uint]bool{}
	for _, c := range cats {
		ids[c.EntityID] = true
	}
	if !ids[c1.EntityID] || !ids[c2.EntityID] {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

func TestCategoryRepository_GetByIDsWithAttributesAndFlat(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)

	cat := seedCategory(t, db, 0, "1", 0, "Flat Cat")
	if err := db.Create(&categoryEntity.CategoryVarchar{
		AttributeID: 119, StoreID: 0, EntityID: cat.EntityID, Value: "flat-cat",
	}).Error; err != nil {
		t.Fatalf("create url_key: %v", err)
	}

	cats, flats, err := repo.GetByIDsWithAttributesAndFlat([]uint{cat.EntityID}, 0)
	if err != nil {
		t.Fatalf("GetByIDsWithAttributesAndFlat: %v", err)
	}
	if len(cats) != 1 || len(flats) != 1 {
		t.Fatalf("expected 1 each, got cats=%d flats=%d", len(cats), len(flats))
	}
	if nameAttr, ok := flats[0]["name"]; ok {
		if nameAttr["value"] != "Flat Cat" {
			t.Errorf("name = %v, want Flat Cat", nameAttr["value"])
		}
	} else {
		t.Error("name missing from flat attrs")
	}
	if urlAttr, ok := flats[0]["url_key"]; ok {
		if urlAttr["value"] != "flat-cat" {
			t.Errorf("url_key = %v, want flat-cat", urlAttr["value"])
		}
	} else {
		t.Error("url_key missing from flat attrs")
	}
}

func TestCategoryRepository_BuildCategoryTree(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)
	repo.InvalidateCache()

	root := seedCategory(t, db, 0, "1", 0, "Root")
	child1 := seedCategory(t, db, root.EntityID, "1/2", 1, "Child 1")
	_ = seedCategory(t, db, root.EntityID, "1/3", 1, "Child 2")
	_ = seedCategory(t, db, child1.EntityID, "1/2/4", 2, "Grandchild")

	tree, err := repo.BuildCategoryTree(0, 0)
	if err != nil {
		t.Fatalf("BuildCategoryTree: %v", err)
	}
	if len(tree) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(tree))
	}
	if tree[0].Category.EntityID != root.EntityID {
		t.Errorf("root entity_id = %d, want %d", tree[0].Category.EntityID, root.EntityID)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected 2 children of root, got %d", len(tree[0].Children))
	}

	var child1Node *categoryRepo.CategoryTreeNode
	for _, ch := range tree[0].Children {
		if ch.Category.EntityID == child1.EntityID {
			child1Node = ch
		}
	}
	if child1Node == nil {
		t.Fatal("child1 not found in tree")
	}
	if len(child1Node.Children) != 1 {
		t.Errorf("expected 1 grandchild, got %d", len(child1Node.Children))
	}
}

func TestCategoryRepository_BuildCategoryTree_Subtree(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)
	repo.InvalidateCache()

	root := seedCategory(t, db, 0, "1", 0, "Root")
	child := seedCategory(t, db, root.EntityID, "1/2", 1, "Child")
	_ = seedCategory(t, db, child.EntityID, "1/2/3", 2, "Grandchild A")
	_ = seedCategory(t, db, child.EntityID, "1/2/4", 2, "Grandchild B")

	subtree, err := repo.BuildCategoryTree(0, child.EntityID)
	if err != nil {
		t.Fatalf("BuildCategoryTree subtree: %v", err)
	}
	if len(subtree) != 2 {
		t.Errorf("expected 2 grandchildren, got %d", len(subtree))
	}
}

func TestCategoryRepository_InvalidateCache(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)

	seedCategory(t, db, 0, "1", 0, "Cached Cat")

	cats1, _ := repo.FetchAllWithAttributes(0)
	if len(cats1) != 1 {
		t.Fatalf("expected 1, got %d", len(cats1))
	}

	seedCategory(t, db, 0, "1/2", 1, "New Cat")

	cats2, _ := repo.FetchAllWithAttributes(0)
	if len(cats2) != 1 {
		t.Log("cache returned stale data as expected")
	}

	repo.InvalidateCache()
	cats3, _ := repo.FetchAllWithAttributes(0)
	if len(cats3) != 2 {
		t.Errorf("after InvalidateCache: expected 2, got %d", len(cats3))
	}
}

func TestCategoryRepository_GetCacheCategory(t *testing.T) {
	db := categoryRepoTestDB(t)
	seedCategoryAttrs(t, db)
	repo := categoryRepo.NewCategoryRepository(db)

	_, ok := repo.GetCacheCategory(0, 0)
	if ok {
		t.Error("expected cache miss on empty repo")
	}

	cat := seedCategory(t, db, 0, "1", 0, "Cache Test")
	_, _ = repo.FetchAllWithAttributes(0)

	allIface, ok := repo.GetCacheCategory(0, 0)
	if !ok {
		t.Fatal("expected cache hit for all categories")
	}
	allMap, ok := allIface.(map[uint]categoryRepo.CategoryWithAttributes)
	if !ok {
		t.Fatal("unexpected type from GetCacheCategory")
	}
	if len(allMap) != 1 {
		t.Errorf("expected 1 cached category, got %d", len(allMap))
	}

	single, ok := repo.GetCacheCategory(0, cat.EntityID)
	if !ok {
		t.Fatal("expected cache hit for specific category")
	}
	cwa, ok := single.(categoryRepo.CategoryWithAttributes)
	if !ok {
		t.Fatal("unexpected type for single category")
	}
	if cwa.Category.EntityID != cat.EntityID {
		t.Errorf("cached entity_id = %d, want %d", cwa.Category.EntityID, cat.EntityID)
	}
}

func TestFlattenCategoryAttributesWithLabels(t *testing.T) {
	nameLabel := "Name"
	attrMeta := map[uint]entity.EavAttribute{
		41:  {AttributeID: 41, AttributeCode: "name", BackendType: "varchar", FrontendLabel: &nameLabel},
		42:  {AttributeID: 42, AttributeCode: "is_active", BackendType: "int"},
		100: {AttributeID: 100, AttributeCode: "description", BackendType: "text"},
	}
	cat := &categoryEntity.Category{
		EntityID: 10,
		Varchars: []categoryEntity.CategoryVarchar{{AttributeID: 41, StoreID: 0, EntityID: 10, Value: "Test Cat"}},
		Ints:     []categoryEntity.CategoryInt{{AttributeID: 42, StoreID: 0, EntityID: 10, Value: 1}},
		Texts:    []categoryEntity.CategoryText{{AttributeID: 100, StoreID: 0, EntityID: 10, Value: "A long description"}},
	}

	flat := categoryRepo.FlattenCategoryAttributesWithLabels(cat, attrMeta)

	if nameAttr, ok := flat["name"]; ok {
		if nameAttr["value"] != "Test Cat" {
			t.Errorf("name value = %v, want Test Cat", nameAttr["value"])
		}
		if nameAttr["label"] != "Name" {
			t.Errorf("name label = %v, want Name", nameAttr["label"])
		}
	} else {
		t.Error("name missing")
	}
	if isActive, ok := flat["is_active"]; ok {
		if isActive["value"] != 1 {
			t.Errorf("is_active value = %v, want 1", isActive["value"])
		}
		if isActive["label"] != "" {
			t.Errorf("is_active label = %v, want empty (no FrontendLabel)", isActive["label"])
		}
	} else {
		t.Error("is_active missing")
	}
	if desc, ok := flat["description"]; ok {
		if desc["value"] != "A long description" {
			t.Errorf("description = %v", desc["value"])
		}
	} else {
		t.Error("description missing")
	}
	if eid, ok := flat["entity_id"]; ok {
		if eid["value"] != uint(10) {
			t.Errorf("entity_id = %v, want 10", eid["value"])
		}
	} else {
		t.Error("entity_id missing")
	}
}
