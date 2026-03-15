package report

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

//go:embed templates/report.html
var reportTemplateFS embed.FS

// Renderer generates HTML and PDF representations of report content.
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer creates a Renderer with the embedded HTML template.
func NewRenderer() *Renderer {
	tmpl := template.Must(
		template.ParseFS(reportTemplateFS, "templates/report.html"),
	)
	return &Renderer{tmpl: tmpl}
}

// RenderHTML generates a standalone HTML string from report content.
func (r *Renderer) RenderHTML(content ReportContent) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, content); err != nil {
		return "", fmt.Errorf("executing report template: %w", err)
	}
	return buf.String(), nil
}

// RenderPDF generates a PDF byte slice from HTML content using chromedp.
func (r *Renderer) RenderPDF(ctx context.Context, htmlContent string) ([]byte, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
	)
	defer allocCancel()

	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	// Set a timeout so we don't hang forever if Chrome is unavailable.
	taskCtx, timeoutCancel := context.WithTimeout(taskCtx, 30*time.Second)
	defer timeoutCancel()

	var pdfBuf []byte

	if err := chromedp.Run(taskCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return fmt.Errorf("getting frame tree: %w", err)
			}
			return page.SetDocumentContent(frameTree.Frame.ID, htmlContent).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithMarginTop(0.5).
				WithMarginBottom(0.5).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			if err != nil {
				return fmt.Errorf("printing to PDF: %w", err)
			}
			pdfBuf = buf
			return nil
		}),
	); err != nil {
		return nil, fmt.Errorf("chromedp PDF render: %w", err)
	}

	return pdfBuf, nil
}
