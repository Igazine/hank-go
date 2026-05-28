package main

import (
	"github.com/Igazine/hank-go"
	"os"
	"path/filepath"
)

type FileResource struct {
	id      string
	content string
	ast     hank.Expr
}

func NewFileResource(path string) *FileResource {
	return &FileResource{id: path}
}

func (f *FileResource) ID() string {
	return f.id
}

func (f *FileResource) Content() string {
	return f.content
}

func (f *FileResource) AST() hank.Expr {
	return f.ast
}

func (f *FileResource) SetAST(ast hank.Expr) {
	f.ast = ast
}

func (f *FileResource) Load() error {
	b, err := os.ReadFile(f.id)
	if err != nil {
		return err
	}
	f.content = string(b)
	return nil
}

func (f *FileResource) Resolve(id string) (hank.Resource, error) {
	path := id
	if !filepath.IsAbs(path) {
		baseDir := filepath.Dir(f.id)
		path = filepath.Join(baseDir, path)
	}

	if filepath.Ext(path) == "" {
		if _, err := os.Stat(path + ".hank"); err == nil {
			path = path + ".hank"
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return NewFileResource(filepath.Clean(absPath)), nil
}
