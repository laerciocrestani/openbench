package pricing

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const pricingURL = SourceGoogle

var (
	modelIDRE     = regexp.MustCompile("`gemini-[a-z0-9.-]+`")
	modelIDHTMLRE = regexp.MustCompile(`(?is)<code[^>]*>(gemini-[a-z0-9.-]+)</code>`)
	h2HTMLRE      = regexp.MustCompile(`(?is)<h2[^>]*>([^<]+)</h2>`)
	standardHTMLRE = regexp.MustCompile(`(?is)<h3[^>]*>\s*Standard\s*</h3>`)
	nextH3HTMLRE  = regexp.MustCompile(`(?is)<h3[^>]*>`)
	inputHTMLRE   = regexp.MustCompile(`(?is)Input price[^<]*</td>\s*<td[^>]*>.*?</td>\s*<td[^>]*>([^<]+)`)
	outputHTMLRE  = regexp.MustCompile(`(?is)Output price[^<]*</td>\s*<td[^>]*>.*?</td>\s*<td[^>]*>([^<]+)`)
	priceValueRE  = regexp.MustCompile(`\$([0-9]+(?:\.[0-9]+)?)`)
)

func FetchAndUpdate() (*Store, error) {
	body, err := downloadPricingPage()
	if err != nil {
		return nil, err
	}

	models, err := ParseGeminiPricing(body)
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("nenhum preço Gemini encontrado na página oficial")
	}

	applyAliases(models)

	store := &Store{
		UpdatedAt: time.Now().UTC(),
		Source:    pricingURL,
		Provider:  "gemini",
		Models:    models,
	}

	if err := Save(*store); err != nil {
		return nil, err
	}
	return store, nil
}

func downloadPricingPage() (string, error) {
	req, err := http.NewRequest(http.MethodGet, pricingURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "gitai-pricing/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("buscar página de preços: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("página de preços retornou %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseGeminiPricing(content string) (map[string]ModelPrice, error) {
	if looksLikeHTML(content) {
		return ParseGeminiPricingHTML(content)
	}
	return parseGeminiPricingMarkdown(content)
}

func looksLikeHTML(content string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(content))
	return strings.HasPrefix(trimmed, "<!doctype") || strings.HasPrefix(trimmed, "<html")
}

func parseGeminiPricingMarkdown(content string) (map[string]ModelPrice, error) {
	content = strings.TrimPrefix(content, "## ")
	sections := strings.Split(content, "\n## ")
	models := make(map[string]ModelPrice)

	for _, section := range sections {
		if !strings.HasPrefix(section, "Gemini") {
			continue
		}

		modelID := firstModelID(section)
		if modelID == "" {
			continue
		}

		standard := extractStandardBlock(section)
		if standard == "" {
			continue
		}

		input, okIn := parseTablePrice(standard, "Input price")
		output, okOut := parseTablePrice(standard, "Output price")
		if !okIn || !okOut {
			continue
		}

		models[modelID] = ModelPrice{
			InputPer1M:  input,
			OutputPer1M: output,
		}
	}

	return models, nil
}

func ParseGeminiPricingHTML(html string) (map[string]ModelPrice, error) {
	models := make(map[string]ModelPrice)
	locations := h2HTMLRE.FindAllStringSubmatchIndex(html, -1)

	for i, loc := range locations {
		title := html[loc[2]:loc[3]]
		if !strings.Contains(title, "Gemini") {
			continue
		}

		end := len(html)
		if i+1 < len(locations) {
			end = locations[i+1][0]
		}
		chunk := html[loc[1]:end]

		codeMatch := modelIDHTMLRE.FindStringSubmatch(chunk)
		if len(codeMatch) < 2 {
			continue
		}
		modelID := codeMatch[1]

		block := chunk
		if stdLoc := standardHTMLRE.FindStringIndex(chunk); stdLoc != nil {
			block = chunk[stdLoc[1]:]
			if next := nextH3HTMLRE.FindStringIndex(block); next != nil {
				block = block[:next[0]]
			}
		}

		inMatch := inputHTMLRE.FindStringSubmatch(block)
		outMatch := outputHTMLRE.FindStringSubmatch(block)
		if len(inMatch) < 2 || len(outMatch) < 2 {
			continue
		}

		input, okIn := firstDollarAmount(strings.TrimSpace(inMatch[1]))
		output, okOut := firstDollarAmount(strings.TrimSpace(outMatch[1]))
		if !okIn || !okOut {
			continue
		}

		models[modelID] = ModelPrice{
			InputPer1M:  input,
			OutputPer1M: output,
		}
	}

	return models, nil
}

func firstModelID(section string) string {
	match := modelIDRE.FindString(section)
	if match == "" {
		return ""
	}
	return strings.Trim(match, "`")
}

func extractStandardBlock(section string) string {
	idx := strings.Index(section, "### Standard")
	if idx < 0 {
		return ""
	}
	block := section[idx+len("### Standard"):]
	if end := strings.Index(block, "\n### "); end >= 0 {
		block = block[:end]
	}
	if end := strings.Index(block, "\n## "); end >= 0 {
		block = block[:end]
	}
	return block
}

func parseTablePrice(block, rowPrefix string) (float64, bool) {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if !strings.Contains(line, rowPrefix) {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		paidTier := strings.TrimSpace(parts[3])
		if strings.Contains(strings.ToLower(paidTier), "not available") {
			return 0, false
		}
		return firstDollarAmount(paidTier)
	}
	return 0, false
}

func firstDollarAmount(s string) (float64, bool) {
	match := priceValueRE.FindStringSubmatch(s)
	if len(match) < 2 {
		return 0, false
	}
	var value float64
	if _, err := fmt.Sscanf(match[1], "%f", &value); err != nil {
		return 0, false
	}
	return value, true
}

func applyAliases(models map[string]ModelPrice) {
	aliases := map[string]string{
		"gemini-3-flash": "gemini-3-flash-preview",
		"gemini-3.1-pro": "gemini-3.1-pro-preview",
	}
	for alias, source := range aliases {
		if p, ok := models[source]; ok {
			models[alias] = p
		}
	}
}
