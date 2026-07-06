package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/laerciocrestani/gitai/internal/pricing"
	"github.com/laerciocrestani/gitai/internal/ui"
)

func RunPricingUpdate() error {
	sess := ui.New("pricing", false)
	sess.Header()

	var store *pricing.Store
	if err := sess.Step("Fetching Gemini pricing", func() error {
		var err error
		store, err = pricing.FetchAndUpdate()
		return err
	}); err != nil {
		return err
	}

	path, _ := pricing.StorePath()
	sess.Detail(fmt.Sprintf("%d modelos salvos em %s", len(store.Models), path))
	sess.Detail(fmt.Sprintf("Fonte: %s", store.Source))
	sess.Detail(fmt.Sprintf("Atualizado: %s", store.UpdatedAt.Format("2006-01-02 15:04 UTC")))

	printPricingTable(sess, store)
	sess.Success("Pricing updated ✨")
	return nil
}

func RunPricingShow() error {
	sess := ui.New("pricing", false)
	sess.Header()

	store, err := pricing.Load()
	if err != nil {
		return err
	}
	if store == nil || len(store.Models) == 0 {
		sess.Info("Nenhum preço salvo. Execute: gitai pricing update")
		return nil
	}

	sess.Detail(fmt.Sprintf("Atualizado: %s", store.UpdatedAt.Format("2006-01-02 15:04 UTC")))
	sess.Detail(fmt.Sprintf("Fonte: %s", store.Source))
	printPricingTable(sess, store)
	return nil
}

func RunPricingReport() error {
	return RunReport(ReportOptions{All: true})
}

func printPricingTable(sess *ui.Session, store *pricing.Store) {
	models := make([]string, 0, len(store.Models))
	for m := range store.Models {
		models = append(models, m)
	}
	sort.Strings(models)

	sess.Section("Preços Gemini (Standard, USD / 1M tokens)")
	for _, model := range models {
		p := store.Models[model]
		sess.Bullet(fmt.Sprintf("%s — entrada $%.4f · saída $%.4f",
			model, p.InputPer1M, p.OutputPer1M))
	}

	known := []string{
		"gemini-2.5-flash-lite", "gemini-2.5-flash", "gemini-2.5-pro",
		"gemini-3.1-flash-lite", "gemini-3-flash", "gemini-3.1-pro",
	}
	var missing []string
	for _, m := range known {
		if _, ok := store.Models[m]; !ok {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		sess.Info("Modelos sem preço na fonte (usando fallback): " + strings.Join(missing, ", "))
	}
}
