package pdfmcp

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMathpixOCR_ExtractText(t *testing.T) {
	// Create a simple test image (1x1 PNG)
	testImageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x5C, 0xC2, 0x8A, 0x8E, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	mathpix := NewMathpixOCR("test_app_id", "test_app_key", []string{"en"})
	ctx := context.Background()

	result, err := mathpix.ExtractText(ctx, testImageData)

	// Note: This test will fail without valid Mathpix credentials
	// In that case, we expect an error
	if err != nil {
		t.Logf("Mathpix not available (expected without credentials): %v", err)
		return
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Engine != "mathpix" {
		t.Errorf("Expected engine 'mathpix', got '%s'", result.Engine)
	}

	if result.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}
}

func TestMathpixOCR_ExtractStructuredText(t *testing.T) {
	testImageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x5C, 0xC2, 0x8A, 0x8E, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	mathpix := NewMathpixOCR("test_app_id", "test_app_key", []string{"en"})
	ctx := context.Background()

	result, err := mathpix.ExtractStructuredText(ctx, testImageData, "typed")

	if err != nil {
		t.Logf("Mathpix not available (expected without credentials): %v", err)
		return
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Engine != "mathpix" {
		t.Errorf("Expected engine 'mathpix', got '%s'", result.Engine)
	}

	if len(result.Blocks) == 0 {
		t.Error("Expected at least one text block")
	}
}

func TestMathpixOCR_GetEngineInfo(t *testing.T) {
	mathpix := NewMathpixOCR("test_app_id", "test_app_key", []string{"en", "fr"})
	info := mathpix.GetEngineInfo()

	if info.Name != "Mathpix" {
		t.Errorf("Expected name 'Mathpix', got '%s'", info.Name)
	}

	if info.IsLocal {
		t.Error("Expected Mathpix to not be local")
	}

	if !info.RequiresAuth {
		t.Error("Expected Mathpix to require auth")
	}

	if len(info.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(info.Languages))
	}

	expectedFeatures := []string{"text_extraction", "math_recognition", "table_extraction", "high_accuracy"}
	for _, feature := range expectedFeatures {
		found := false
		for _, infoFeature := range info.Features {
			if infoFeature == feature {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected feature '%s' not found", feature)
		}
	}
}

func TestMathpixOCR_ParseMarkdownToBlocks(t *testing.T) {
	mathpix := NewMathpixOCR("test_app_id", "test_app_key", []string{"en"})

	markdown := `# Title
This is a paragraph.

| Column 1 | Column 2 |
|----------|----------|
| Cell 1   | Cell 2   |

$$E = mc^2$$`

	blocks := mathpix.parseMarkdownToBlocks(markdown)

	if len(blocks) == 0 {
		t.Fatal("Expected blocks, got none")
	}

	// Check for different block types
	hasHeading := false
	hasParagraph := false
	hasTable := false
	hasMath := false

	for _, block := range blocks {
		switch block.Type {
		case "heading":
			hasHeading = true
		case "paragraph":
			hasParagraph = true
		case "table_row":
			hasTable = true
		case "math":
			hasMath = true
		}
	}

	if !hasHeading {
		t.Error("Expected heading block")
	}
	if !hasParagraph {
		t.Error("Expected paragraph block")
	}
	if !hasTable {
		t.Error("Expected table block")
	}
	if !hasMath {
		t.Error("Expected math block")
	}
}

func TestMockOCR_ExtractText(t *testing.T) {
	testImageData := []byte("test image data")
	mockOCR := NewMockOCR([]string{"eng"})
	ctx := context.Background()

	result, err := mockOCR.ExtractText(ctx, testImageData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Engine != "mock" {
		t.Errorf("Expected engine 'mock', got '%s'", result.Engine)
	}

	if result.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", result.Confidence)
	}

	if !strings.Contains(result.Text, "mock OCR text") {
		t.Error("Expected mock text to contain 'mock OCR text'")
	}
}

func TestMockOCR_ExtractStructuredText(t *testing.T) {
	testImageData := []byte("test image data")
	mockOCR := NewMockOCR([]string{"eng"})
	ctx := context.Background()

	result, err := mockOCR.ExtractStructuredText(ctx, testImageData, "typed")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Blocks) == 0 {
		t.Error("Expected at least one text block")
	}

	if result.Layout.PageWidth != 600 {
		t.Errorf("Expected page width 600, got %d", result.Layout.PageWidth)
	}

	if result.Layout.Orientation != "portrait" {
		t.Errorf("Expected orientation 'portrait', got '%s'", result.Layout.Orientation)
	}
}

func TestOCRManager_RegisterEngine(t *testing.T) {
	manager := NewOCRManager()
	mockOCR := NewMockOCR([]string{"eng"})

	manager.RegisterEngine("test_mock", mockOCR)

	engine, err := manager.GetEngine("test_mock")
	if err != nil {
		t.Fatalf("Failed to get registered engine: %v", err)
	}

	if engine != mockOCR {
		t.Error("Retrieved engine is not the same as registered engine")
	}
}

func TestOCRManager_SetDefaultEngine(t *testing.T) {
	manager := NewOCRManager()
	mockOCR := NewMockOCR([]string{"eng"})

	manager.RegisterEngine("test_mock", mockOCR)
	err := manager.SetDefaultEngine("test_mock")
	if err != nil {
		t.Fatalf("Failed to set default engine: %v", err)
	}

	// Test getting engine with empty name (should return default)
	engine, err := manager.GetEngine("")
	if err != nil {
		t.Fatalf("Failed to get default engine: %v", err)
	}

	if engine != mockOCR {
		t.Error("Default engine is not the expected engine")
	}
}

func TestOCRManager_SetDefaultEngine_NotFound(t *testing.T) {
	manager := NewOCRManager()

	err := manager.SetDefaultEngine("nonexistent")
	if err == nil {
		t.Error("Expected error when setting nonexistent engine as default")
	}
}

func TestOCRManager_GetEngine_NotFound(t *testing.T) {
	manager := NewOCRManager()

	_, err := manager.GetEngine("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent engine")
	}
}

func TestOCRManager_ListEngines(t *testing.T) {
	manager := NewOCRManager()
	mockOCR1 := NewMockOCR([]string{"eng"})
	mockOCR2 := NewMockOCR([]string{"fra"})

	manager.RegisterEngine("mock1", mockOCR1)
	manager.RegisterEngine("mock2", mockOCR2)

	engines := manager.ListEngines()

	if len(engines) != 2 {
		t.Errorf("Expected 2 engines, got %d", len(engines))
	}

	if _, exists := engines["mock1"]; !exists {
		t.Error("Expected 'mock1' engine in list")
	}

	if _, exists := engines["mock2"]; !exists {
		t.Error("Expected 'mock2' engine in list")
	}
}

func TestOCRManager_SuggestEngine(t *testing.T) {
	manager := NewOCRManager()
	mathpix := NewMathpixOCR("test_id", "test_key", []string{"en"})
	mockOCR := NewMockOCR([]string{"eng"})

	manager.RegisterEngine("mathpix", mathpix)
	manager.RegisterEngine("mock", mockOCR)

	tests := []struct {
		documentType string
		imageSize    int
		expected     string
	}{
		{"typed", 1024, "mathpix"},
		{"research", 1024, "mathpix"},
		{"handwritten", 1024, "mathpix"},
		{"mixed", 1024, "mathpix"},
		{"typed", 6 * 1024 * 1024, "mathpix"}, // Large file, still prefer mathpix
	}

	for _, test := range tests {
		result := manager.SuggestEngine(test.documentType, test.imageSize)
		if result != test.expected {
			t.Errorf("For document type '%s' and size %d, expected '%s', got '%s'",
				test.documentType, test.imageSize, test.expected, result)
		}
	}
}

func TestOCRManager_SuggestEngine_Fallback(t *testing.T) {
	manager := NewOCRManager()
	mockOCR := NewMockOCR([]string{"eng"})
	manager.RegisterEngine("mock", mockOCR)

	// Without mathpix, should fallback to mock
	result := manager.SuggestEngine("typed", 1024)
	if result != "mock" {
		t.Errorf("Expected fallback to 'mock', got '%s'", result)
	}
}

func TestOCRManager_ExtractTextWithBestEngine(t *testing.T) {
	manager := NewOCRManager()
	mockOCR := NewMockOCR([]string{"eng"})
	manager.RegisterEngine("mock", mockOCR)
	if err := manager.SetDefaultEngine("mock"); err != nil {
		t.Fatalf("Failed to set default OCR engine: %v", err)
	}

	testImageData := []byte("test image data")
	ctx := context.Background()

	result, err := manager.ExtractTextWithBestEngine(ctx, testImageData, "typed")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Engine != "mock" {
		t.Errorf("Expected engine 'mock', got '%s'", result.Engine)
	}
}

func TestOCRManager_ExtractStructuredTextWithBestEngine(t *testing.T) {
	manager := NewOCRManager()
	mockOCR := NewMockOCR([]string{"eng"})
	manager.RegisterEngine("mock", mockOCR)
	if err := manager.SetDefaultEngine("mock"); err != nil {
		t.Fatalf("Failed to set default OCR engine: %v", err)
	}

	testImageData := []byte("test image data")
	ctx := context.Background()

	result, err := manager.ExtractStructuredTextWithBestEngine(ctx, testImageData, "typed")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Engine != "mock" {
		t.Errorf("Expected engine 'mock', got '%s'", result.Engine)
	}

	if len(result.Blocks) == 0 {
		t.Error("Expected at least one text block")
	}
}

func TestOCRResult_Structure(t *testing.T) {
	result := OCRResult{
		Text:           "Test text",
		Confidence:     0.95,
		Language:       "eng",
		ProcessingTime: time.Second,
		Engine:         "test",
	}

	if result.Text != "Test text" {
		t.Errorf("Expected text 'Test text', got '%s'", result.Text)
	}

	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
	}

	if result.Language != "eng" {
		t.Errorf("Expected language 'eng', got '%s'", result.Language)
	}

	if result.ProcessingTime != time.Second {
		t.Errorf("Expected processing time 1s, got %v", result.ProcessingTime)
	}

	if result.Engine != "test" {
		t.Errorf("Expected engine 'test', got '%s'", result.Engine)
	}
}

func TestStructuredOCRResult_Structure(t *testing.T) {
	blocks := []TextBlock{
		{
			Text:       "Test block",
			Confidence: 0.9,
			BoundingBox: BoundingBox{
				X:      10,
				Y:      20,
				Width:  100,
				Height: 30,
			},
			Type: "paragraph",
		},
	}

	layout := LayoutInfo{
		PageWidth:   800,
		PageHeight:  1000,
		Orientation: "portrait",
		ColumnCount: 1,
		HasTables:   false,
		HasDiagrams: false,
	}

	result := StructuredOCRResult{
		OCRResult: OCRResult{
			Text:           "Test text",
			Confidence:     0.95,
			Language:       "eng",
			ProcessingTime: time.Second,
			Engine:         "test",
		},
		Blocks: blocks,
		Layout: layout,
	}

	if len(result.Blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(result.Blocks))
	}

	if result.Blocks[0].Text != "Test block" {
		t.Errorf("Expected block text 'Test block', got '%s'", result.Blocks[0].Text)
	}

	if result.Layout.PageWidth != 800 {
		t.Errorf("Expected page width 800, got %d", result.Layout.PageWidth)
	}
}

func TestBoundingBox_Structure(t *testing.T) {
	bbox := BoundingBox{
		X:      10,
		Y:      20,
		Width:  100,
		Height: 50,
	}

	if bbox.X != 10 {
		t.Errorf("Expected X 10, got %d", bbox.X)
	}

	if bbox.Y != 20 {
		t.Errorf("Expected Y 20, got %d", bbox.Y)
	}

	if bbox.Width != 100 {
		t.Errorf("Expected Width 100, got %d", bbox.Width)
	}

	if bbox.Height != 50 {
		t.Errorf("Expected Height 50, got %d", bbox.Height)
	}
}

func TestOCREngineInfo_Structure(t *testing.T) {
	info := OCREngineInfo{
		Name:         "Test Engine",
		Version:      "1.0",
		Languages:    []string{"eng", "fra"},
		Features:     []string{"text_extraction", "layout_analysis"},
		IsLocal:      true,
		RequiresAuth: false,
	}

	if info.Name != "Test Engine" {
		t.Errorf("Expected name 'Test Engine', got '%s'", info.Name)
	}

	if info.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", info.Version)
	}

	if len(info.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(info.Languages))
	}

	if len(info.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(info.Features))
	}

	if !info.IsLocal {
		t.Error("Expected IsLocal to be true")
	}

	if info.RequiresAuth {
		t.Error("Expected RequiresAuth to be false")
	}
}

// Benchmark tests
func BenchmarkMathpixOCR_ExtractText(b *testing.B) {
	testImageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x5C, 0xC2, 0x8A, 0x8E, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	mathpix := NewMathpixOCR("test_app_id", "test_app_key", []string{"en"})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mathpix.ExtractText(ctx, testImageData)
		if err != nil {
			b.Skip("Mathpix not available")
		}
	}
}

func BenchmarkMockOCR_ExtractText(b *testing.B) {
	testImageData := []byte("test image data")
	mockOCR := NewMockOCR([]string{"eng"})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockOCR.ExtractText(ctx, testImageData)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

func BenchmarkOCRManager_SuggestEngine(b *testing.B) {
	manager := NewOCRManager()
	mathpix := NewMathpixOCR("test_id", "test_key", []string{"en"})
	mockOCR := NewMockOCR([]string{"eng"})

	manager.RegisterEngine("mathpix", mathpix)
	manager.RegisterEngine("mock", mockOCR)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.SuggestEngine("typed", 1024)
	}
}
