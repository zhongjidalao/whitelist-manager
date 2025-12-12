package templates

import (
	"embed"
	"html/template"

	"github.com/gin-gonic/gin"
)

//go:embed *.html
var FS embed.FS

// LoadTemplates loads embedded HTML templates into Gin engine
func LoadTemplates(engine *gin.Engine) error {
	tmpl, err := template.ParseFS(FS, "*.html")
	if err != nil {
		return err
	}
	engine.SetHTMLTemplate(tmpl)
	return nil
}
