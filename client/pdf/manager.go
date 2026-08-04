package pdf

import (
	"fmt"
	"image"

	"github.com/gen2brain/go-fitz"
)

type Manager struct {
	doc *fitz.Document
}

func NewManager(filePath string) (*Manager, error) {
	doc, err := fitz.New(filePath)
	if err != nil {
		return nil, err
	}
	return &Manager{doc: doc}, nil
}

func (m *Manager) Close() {
	if m.doc != nil {
		m.doc.Close()
	}
}

func (m *Manager) GetPageCount() int {
	if m.doc != nil {
		return m.doc.NumPage()
	}
	return 0
}

func (m *Manager) GetPageImage(pageIndex int) (image.Image, error) {
	if m.doc == nil {
		return nil, fmt.Errorf("no document loaded")
	}
	if pageIndex < 0 || pageIndex >= m.doc.NumPage() {
		return nil, fmt.Errorf("page index out of range")
	}
	return m.doc.Image(pageIndex)
}
