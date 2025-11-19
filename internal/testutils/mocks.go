package testutils

import (
	"time"

	"github.com/rockstaedt/swimmate/internal/models"
)

// MockSwimModel is a mock implementation of models.SwimModel for testing
type MockSwimModel struct {
	GetFunc           func() (*models.Swim, error)
	GetAllFunc        func(userId int) ([]*models.Swim, error)
	GetPaginatedFunc  func(userId int, limit int, offset int) ([]*models.Swim, error)
	InsertFunc        func(date time.Time, distanceM int, assessment int, userId int) error
	SummarizeFunc     func(userId int) *models.SwimSummary
}

func (m *MockSwimModel) Get() (*models.Swim, error) {
	if m.GetFunc != nil {
		return m.GetFunc()
	}
	return &models.Swim{}, nil
}

func (m *MockSwimModel) GetAll(userId int) ([]*models.Swim, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(userId)
	}
	return []*models.Swim{}, nil
}

func (m *MockSwimModel) GetPaginated(userId int, limit int, offset int) ([]*models.Swim, error) {
	if m.GetPaginatedFunc != nil {
		return m.GetPaginatedFunc(userId, limit, offset)
	}
	return []*models.Swim{}, nil
}

func (m *MockSwimModel) Insert(date time.Time, distanceM int, assessment int, userId int) error {
	if m.InsertFunc != nil {
		return m.InsertFunc(date, distanceM, assessment, userId)
	}
	return nil
}

func (m *MockSwimModel) Summarize(userId int) *models.SwimSummary {
	if m.SummarizeFunc != nil {
		return m.SummarizeFunc(userId)
	}
	return &models.SwimSummary{
		YearMap: make(map[int]models.YearMap),
	}
}

// MockUserModel is a mock implementation of models.UserModel for testing
type MockUserModel struct {
	AuthenticateFunc func(username, password string) (int, error)
}

func (m *MockUserModel) Authenticate(username, password string) (int, error) {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(username, password)
	}
	return 0, models.ErrInvalidCredentials
}
