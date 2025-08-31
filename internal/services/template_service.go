package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

// TemplateService handles template-related operations
type TemplateService interface {
	CreateTemplate(template *models.Template) error
	GetTemplateByID(id uint) (*models.Template, error)
	ListTemplates(chainType, keyword string, limit int) ([]models.Template, error)
	UpdateTemplate(template *models.Template) error
	DeleteTemplate(id uint) error
	DeleteTemplates(ids []uint) (int64, error)
}

type templateService struct {
	db *gorm.DB
}

// NewTemplateService creates a new TemplateService
func NewTemplateService(db *gorm.DB) TemplateService {
	return &templateService{db: db}
}

// CreateTemplate creates a new template
func (s *templateService) CreateTemplate(template *models.Template) error {
	return s.db.Create(template).Error
}

// GetTemplateByID returns a template by its ID
func (s *templateService) GetTemplateByID(id uint) (*models.Template, error) {
	var template models.Template
	err := s.db.First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// ListTemplates returns templates with optional filtering
func (s *templateService) ListTemplates(chainType, keyword string, limit int) ([]models.Template, error) {
	query := s.db.Model(&models.Template{})

	if chainType != "" {
		query = query.Where("chain_type = ?", chainType)
	}

	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var templates []models.Template
	err := query.Find(&templates).Error
	return templates, err
}

// UpdateTemplate updates an existing template
func (s *templateService) UpdateTemplate(template *models.Template) error {
	return s.db.Save(template).Error
}

// DeleteTemplate deletes a template by its ID
func (s *templateService) DeleteTemplate(id uint) error {
	return s.db.Delete(&models.Template{}, id).Error
}

// DeleteTemplates deletes multiple templates by their IDs
func (s *templateService) DeleteTemplates(ids []uint) (int64, error) {
	result := s.db.Delete(&models.Template{}, ids)
	return result.RowsAffected, result.Error
}
