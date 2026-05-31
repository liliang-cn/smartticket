package importexport

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	apperrors "github.com/company/smartticket/internal/errors"
	"github.com/company/smartticket/internal/models"
)

// formatMeta describes the on-disk representation for an export format.
type formatMeta struct {
	contentType string
	ext         string
}

// formatMetaFor returns the content type and file extension for a target format.
func formatMetaFor(f FileType) (formatMeta, bool) {
	switch f {
	case FileTypeJSON:
		return formatMeta{contentType: "application/json", ext: "json"}, true
	case FileTypeCSV:
		return formatMeta{contentType: "text/csv", ext: "csv"}, true
	case FileTypeXML:
		return formatMeta{contentType: "application/xml", ext: "xml"}, true
	case FileTypeMarkdown:
		return formatMeta{contentType: "text/markdown", ext: "md"}, true
	case FileTypeSQLite:
		return formatMeta{contentType: "application/octet-stream", ext: "db"}, true
	}
	return formatMeta{}, false
}

// ContentTypeForExt maps a file extension (with or without leading dot) to a
// content type for download responses.
func ContentTypeForExt(ext string) string {
	switch ext {
	case ".json", "json":
		return "application/json"
	case ".csv", "csv":
		return "text/csv"
	case ".xml", "xml":
		return "application/xml"
	case ".md", "md", ".markdown", "markdown":
		return "text/markdown"
	case ".db", "db", ".sqlite", "sqlite", ".sqlite3", "sqlite3":
		return "application/octet-stream"
	}
	return "application/octet-stream"
}

// --- safe/flat view structs (exclude secrets, flatten key fields) ---

type ticketView struct {
	ID             uint   `json:"id" xml:"id"`
	TicketNumber   string `json:"ticket_number" xml:"ticket_number"`
	Title          string `json:"title" xml:"title"`
	Description    string `json:"description" xml:"description"`
	Status         string `json:"status" xml:"status"`
	Priority       string `json:"priority" xml:"priority"`
	Severity       string `json:"severity" xml:"severity"`
	Category       string `json:"category" xml:"category"`
	Type           string `json:"type" xml:"type"`
	CustomerName   string `json:"customer_name" xml:"customer_name"`
	AssignedTo     string `json:"assigned_to" xml:"assigned_to"`
	RequesterName  string `json:"requester_name" xml:"requester_name"`
	RequesterEmail string `json:"requester_email" xml:"requester_email"`
	SLAStatus      string `json:"sla_status" xml:"sla_status"`
	CreatedAt      string `json:"created_at" xml:"created_at"`
}

type knowledgeArticleView struct {
	ID          uint   `json:"id" xml:"id"`
	Title       string `json:"title" xml:"title"`
	Slug        string `json:"slug" xml:"slug"`
	Summary     string `json:"summary" xml:"summary"`
	Content     string `json:"content" xml:"content"`
	Status      string `json:"status" xml:"status"`
	Visibility  string `json:"visibility" xml:"visibility"`
	AccessLevel string `json:"access_level" xml:"access_level"`
	Category    string `json:"category" xml:"category"`
	Views       int    `json:"views" xml:"views"`
	Version     int    `json:"version" xml:"version"`
	CreatedAt   string `json:"created_at" xml:"created_at"`
}

// userView intentionally excludes PasswordHash and any secret material.
type userView struct {
	ID        uint   `json:"id" xml:"id"`
	Email     string `json:"email" xml:"email"`
	Username  string `json:"username" xml:"username"`
	FirstName string `json:"first_name" xml:"first_name"`
	LastName  string `json:"last_name" xml:"last_name"`
	Role      string `json:"role" xml:"role"`
	IsActive  bool   `json:"is_active" xml:"is_active"`
	CreatedAt string `json:"created_at" xml:"created_at"`
}

type productView struct {
	ID           uint   `json:"id" xml:"id"`
	Name         string `json:"name" xml:"name"`
	Code         string `json:"code" xml:"code"`
	Description  string `json:"description" xml:"description"`
	Category     string `json:"category" xml:"category"`
	Version      string `json:"version" xml:"version"`
	Status       string `json:"status" xml:"status"`
	SupportLevel string `json:"support_level" xml:"support_level"`
	CreatedAt    string `json:"created_at" xml:"created_at"`
}

type serviceView struct {
	ID           uint   `json:"id" xml:"id"`
	ProductID    uint   `json:"product_id" xml:"product_id"`
	Name         string `json:"name" xml:"name"`
	Code         string `json:"code" xml:"code"`
	Description  string `json:"description" xml:"description"`
	Type         string `json:"type" xml:"type"`
	Status       string `json:"status" xml:"status"`
	Availability string `json:"availability" xml:"availability"`
	CreatedAt    string `json:"created_at" xml:"created_at"`
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func toTicketViews(rows []models.Ticket) []ticketView {
	out := make([]ticketView, 0, len(rows))
	for _, r := range rows {
		v := ticketView{
			ID:             r.ID,
			TicketNumber:   r.TicketNumber,
			Title:          r.Title,
			Description:    r.Description,
			Status:         r.Status,
			Priority:       r.Priority,
			Severity:       r.Severity,
			Category:       r.Category,
			Type:           r.Type,
			RequesterName:  r.RequesterName,
			RequesterEmail: r.RequesterEmail,
			SLAStatus:      r.SLAStatus,
			CreatedAt:      fmtTime(r.CreatedAt),
		}
		if r.Customer != nil {
			v.CustomerName = r.Customer.Name
		}
		if r.AssignedUser != nil {
			v.AssignedTo = r.AssignedUser.Email
		}
		out = append(out, v)
	}
	return out
}

func toKnowledgeArticleViews(rows []models.KnowledgeArticle) []knowledgeArticleView {
	out := make([]knowledgeArticleView, 0, len(rows))
	for _, r := range rows {
		out = append(out, knowledgeArticleView{
			ID:          r.ID,
			Title:       r.Title,
			Slug:        r.Slug,
			Summary:     r.Summary,
			Content:     r.Content,
			Status:      r.Status,
			Visibility:  r.Visibility,
			AccessLevel: r.AccessLevel,
			Category:    r.Category,
			Views:       r.Views,
			Version:     r.Version,
			CreatedAt:   fmtTime(r.CreatedAt),
		})
	}
	return out
}

func toUserViews(rows []models.User) []userView {
	out := make([]userView, 0, len(rows))
	for _, r := range rows {
		out = append(out, userView{
			ID:        r.ID,
			Email:     r.Email,
			Username:  r.Username,
			FirstName: r.FirstName,
			LastName:  r.LastName,
			Role:      r.Role,
			IsActive:  r.IsActive,
			CreatedAt: fmtTime(r.CreatedAt),
		})
	}
	return out
}

func toProductViews(rows []models.Product) []productView {
	out := make([]productView, 0, len(rows))
	for _, r := range rows {
		out = append(out, productView{
			ID:           r.ID,
			Name:         r.Name,
			Code:         r.Code,
			Description:  r.Description,
			Category:     r.Category,
			Version:      r.Version,
			Status:       r.Status,
			SupportLevel: r.SupportLevel,
			CreatedAt:    fmtTime(r.CreatedAt),
		})
	}
	return out
}

func toServiceViews(rows []models.Service) []serviceView {
	out := make([]serviceView, 0, len(rows))
	for _, r := range rows {
		out = append(out, serviceView{
			ID:           r.ID,
			ProductID:    r.ProductID,
			Name:         r.Name,
			Code:         r.Code,
			Description:  r.Description,
			Type:         r.Type,
			Status:       r.Status,
			Availability: r.Availability,
			CreatedAt:    fmtTime(r.CreatedAt),
		})
	}
	return out
}

// runExport resolves rows for the requested type, serializes them in the
// requested format, and writes the result to disk. On success it populates the
// job's FilePath, TotalRecords, ProcessedRecords and Progress fields (the
// caller persists them). It does not itself save the job.
func (s *Service) runExport(job *models.ImportExportJob, req *ExportRequest) error {
	meta, ok := formatMetaFor(req.TargetFormat)
	if !ok {
		return apperrors.NewValidationError("unsupported export format: " + string(req.TargetFormat))
	}

	// "complete" only supports json/sqlite — a tabular/markdown layout is not
	// meaningful for a multi-entity bundle.
	if req.Type == ExportTypeComplete && req.TargetFormat != FileTypeJSON && req.TargetFormat != FileTypeSQLite {
		return apperrors.NewValidationError("complete export only supports json/sqlite")
	}

	count, payload, err := s.buildPayload(req, meta)
	if err != nil {
		return err
	}

	dir := filepath.Join(s.dataPath, "exports")
	if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
		return apperrors.NewFileError("create export dir", dir, mkErr)
	}
	path := filepath.Join(dir, fmt.Sprintf("export-%d-%s.%s", job.ID, req.Type, meta.ext))

	if writeErr := os.WriteFile(path, payload, 0o640); writeErr != nil {
		return apperrors.NewFileError("write export file", path, writeErr)
	}

	job.FilePath = path
	job.TotalRecords = count
	job.ProcessedRecords = count
	job.Progress = 100
	return nil
}

// buildPayload resolves rows and serializes them, returning the record count
// and the serialized bytes.
func (s *Service) buildPayload(req *ExportRequest, meta formatMeta) (int, []byte, error) {
	switch req.Type {
	case ExportTypeTickets:
		var rows []models.Ticket
		if err := s.db.Preload("Customer").Preload("AssignedUser").Find(&rows).Error; err != nil {
			return 0, nil, apperrors.NewInternalError("failed to query tickets: %w", err)
		}
		views := toTicketViews(rows)
		b, err := serialize(req.TargetFormat, views, "tickets", "ticket")
		return len(views), b, err

	case ExportTypeKnowledgeArticles:
		var rows []models.KnowledgeArticle
		if err := s.db.Find(&rows).Error; err != nil {
			return 0, nil, apperrors.NewInternalError("failed to query knowledge articles: %w", err)
		}
		views := toKnowledgeArticleViews(rows)
		b, err := serialize(req.TargetFormat, views, "knowledge_articles", "article")
		return len(views), b, err

	case ExportTypeUsers:
		var rows []models.User
		if err := s.db.Find(&rows).Error; err != nil {
			return 0, nil, apperrors.NewInternalError("failed to query users: %w", err)
		}
		views := toUserViews(rows)
		b, err := serialize(req.TargetFormat, views, "users", "user")
		return len(views), b, err

	case ExportTypeProducts:
		var rows []models.Product
		if err := s.db.Find(&rows).Error; err != nil {
			return 0, nil, apperrors.NewInternalError("failed to query products: %w", err)
		}
		views := toProductViews(rows)
		b, err := serialize(req.TargetFormat, views, "products", "product")
		return len(views), b, err

	case ExportTypeServices:
		var rows []models.Service
		if err := s.db.Find(&rows).Error; err != nil {
			return 0, nil, apperrors.NewInternalError("failed to query services: %w", err)
		}
		views := toServiceViews(rows)
		b, err := serialize(req.TargetFormat, views, "services", "service")
		return len(views), b, err

	case ExportTypeComplete:
		return s.buildCompletePayload(req)

	default:
		return 0, nil, apperrors.NewValidationError("unsupported export type: " + string(req.Type))
	}
}

// completeBundle aggregates all exportable entities for a "complete" export.
type completeBundle struct {
	Tickets           []ticketView           `json:"tickets"`
	KnowledgeArticles []knowledgeArticleView `json:"knowledge_articles"`
	Users             []userView             `json:"users"`
	Products          []productView          `json:"products"`
	Services          []serviceView          `json:"services"`
}

func (s *Service) buildCompletePayload(req *ExportRequest) (int, []byte, error) {
	var tickets []models.Ticket
	if err := s.db.Preload("Customer").Preload("AssignedUser").Find(&tickets).Error; err != nil {
		return 0, nil, apperrors.NewInternalError("failed to query tickets: %w", err)
	}
	var articles []models.KnowledgeArticle
	if err := s.db.Find(&articles).Error; err != nil {
		return 0, nil, apperrors.NewInternalError("failed to query knowledge articles: %w", err)
	}
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return 0, nil, apperrors.NewInternalError("failed to query users: %w", err)
	}
	var products []models.Product
	if err := s.db.Find(&products).Error; err != nil {
		return 0, nil, apperrors.NewInternalError("failed to query products: %w", err)
	}
	var services []models.Service
	if err := s.db.Find(&services).Error; err != nil {
		return 0, nil, apperrors.NewInternalError("failed to query services: %w", err)
	}

	bundle := completeBundle{
		Tickets:           toTicketViews(tickets),
		KnowledgeArticles: toKnowledgeArticleViews(articles),
		Users:             toUserViews(users),
		Products:          toProductViews(products),
		Services:          toServiceViews(services),
	}
	count := len(bundle.Tickets) + len(bundle.KnowledgeArticles) + len(bundle.Users) + len(bundle.Products) + len(bundle.Services)

	switch req.TargetFormat {
	case FileTypeJSON:
		b, err := json.MarshalIndent(bundle, "", "  ")
		if err != nil {
			return 0, nil, apperrors.NewInternalError("failed to marshal complete bundle: %w", err)
		}
		return count, b, nil
	case FileTypeSQLite:
		// Not implemented: copying the live DB file is out of scope here.
		return 0, nil, apperrors.NewValidationError("sqlite export not supported for complete")
	default:
		return 0, nil, apperrors.NewValidationError("complete export only supports json/sqlite")
	}
}

// serialize converts a slice of view structs to the requested format. The
// argument must be a slice. rootElem/itemElem name the XML wrapper elements.
func serialize(format FileType, slice interface{}, rootElem, itemElem string) ([]byte, error) {
	switch format {
	case FileTypeJSON:
		b, err := json.MarshalIndent(slice, "", "  ")
		if err != nil {
			return nil, apperrors.NewInternalError("failed to marshal json: %w", err)
		}
		return b, nil
	case FileTypeCSV:
		return marshalCSV(slice)
	case FileTypeMarkdown:
		return marshalMarkdown(slice)
	case FileTypeXML:
		return marshalXML(slice, rootElem, itemElem)
	case FileTypeSQLite:
		return nil, apperrors.NewValidationError("sqlite export not supported for tabular types; use json/csv/xml/markdown")
	default:
		return nil, apperrors.NewValidationError("unsupported export format: " + string(format))
	}
}

// structFields returns the JSON-tag-derived header names and the field index
// order for the element type of the given slice.
func structFields(elem reflect.Type) ([]string, []int) {
	headers := make([]string, 0, elem.NumField())
	idx := make([]int, 0, elem.NumField())
	for i := 0; i < elem.NumField(); i++ {
		f := elem.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		name := f.Name
		if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
			if comma := indexByte(tag, ','); comma >= 0 {
				name = tag[:comma]
			} else {
				name = tag
			}
		}
		headers = append(headers, name)
		idx = append(idx, i)
	}
	return headers, idx
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func cellString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func marshalCSV(slice interface{}) ([]byte, error) {
	sv := reflect.ValueOf(slice)
	if sv.Kind() != reflect.Slice {
		return nil, apperrors.NewInternalError("csv export expects a slice", nil)
	}
	elem := sv.Type().Elem()
	headers, idx := structFields(elem)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(headers); err != nil {
		return nil, apperrors.NewInternalError("failed to write csv header: %w", err)
	}
	for i := 0; i < sv.Len(); i++ {
		row := sv.Index(i)
		rec := make([]string, len(idx))
		for j, fi := range idx {
			rec[j] = cellString(row.Field(fi))
		}
		if err := w.Write(rec); err != nil {
			return nil, apperrors.NewInternalError("failed to write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, apperrors.NewInternalError("failed to flush csv: %w", err)
	}
	return buf.Bytes(), nil
}

func mdEscape(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '|':
			b.WriteString("\\|")
		case '\n', '\r':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func marshalMarkdown(slice interface{}) ([]byte, error) {
	sv := reflect.ValueOf(slice)
	if sv.Kind() != reflect.Slice {
		return nil, apperrors.NewInternalError("markdown export expects a slice", nil)
	}
	elem := sv.Type().Elem()
	headers, idx := structFields(elem)

	var buf bytes.Buffer
	buf.WriteString("| ")
	for i, h := range headers {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(mdEscape(h))
	}
	buf.WriteString(" |\n|")
	for range headers {
		buf.WriteString(" --- |")
	}
	buf.WriteString("\n")
	for i := 0; i < sv.Len(); i++ {
		row := sv.Index(i)
		buf.WriteString("| ")
		for j, fi := range idx {
			if j > 0 {
				buf.WriteString(" | ")
			}
			buf.WriteString(mdEscape(cellString(row.Field(fi))))
		}
		buf.WriteString(" |\n")
	}
	return buf.Bytes(), nil
}

func marshalXML(slice interface{}, rootElem, itemElem string) ([]byte, error) {
	sv := reflect.ValueOf(slice)
	if sv.Kind() != reflect.Slice {
		return nil, apperrors.NewInternalError("xml export expects a slice", nil)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString("<" + rootElem + ">\n")
	for i := 0; i < sv.Len(); i++ {
		b, err := xml.MarshalIndent(sv.Index(i).Interface(), "  ", "  ")
		if err != nil {
			return nil, apperrors.NewInternalError("failed to marshal xml: %w", err)
		}
		// Replace the default struct element name with itemElem.
		b = bytes.Replace(b, []byte("<"+sv.Type().Elem().Name()+">"), []byte("<"+itemElem+">"), 1)
		b = bytes.Replace(b, []byte("</"+sv.Type().Elem().Name()+">"), []byte("</"+itemElem+">"), 1)
		buf.Write([]byte("  "))
		buf.Write(b)
		buf.WriteString("\n")
	}
	buf.WriteString("</" + rootElem + ">\n")
	return buf.Bytes(), nil
}
