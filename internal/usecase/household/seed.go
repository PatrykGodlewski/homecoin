package household

import (
	"context"

	"github.com/godlew/homecoin/internal/domain/entity"
	"github.com/godlew/homecoin/internal/domain/repository"
)

var defaultCategories = []struct {
	Name    string
	Icon    string
	Color   string
	IsFixed bool
}{
	{Name: "Rent", Icon: "home", Color: "#6366F1", IsFixed: true},
	{Name: "Groceries", Icon: "cart", Color: "#22C55E", IsFixed: false},
	{Name: "Utilities", Icon: "bolt", Color: "#F59E0B", IsFixed: true},
	{Name: "Transport", Icon: "car", Color: "#3B82F6", IsFixed: false},
	{Name: "Entertainment", Icon: "film", Color: "#EC4899", IsFixed: false},
	{Name: "Dining Out", Icon: "utensils", Color: "#EF4444", IsFixed: false},
	{Name: "Healthcare", Icon: "heart", Color: "#14B8A6", IsFixed: false},
	{Name: "Savings", Icon: "piggy-bank", Color: "#8B5CF6", IsFixed: false},
}

func seedDefaultCategories(ctx context.Context, categories repository.CategoryRepository, householdID string) error {
	for _, dc := range defaultCategories {
		icon := dc.Icon
		color := dc.Color
		c := &entity.Category{
			HouseholdID: householdID,
			Name:        dc.Name,
			Icon:        &icon,
			Color:       &color,
			IsFixed:     dc.IsFixed,
		}
		if err := categories.Create(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
