package taxes

type upsertTaxRuleRequest struct {
	Name               string  `json:"name" validate:"required,min=2,max=120"`
	Percent            float64 `json:"percent" validate:"required,gte=0,lte=100"`
	Level              int32   `json:"level" validate:"required,gte=1"`
	PrependBeforeFirst bool    `json:"prepend_before_first"`
}

type reorderTaxRulesRequest struct {
	OrderedIDs []string `json:"ordered_ids" validate:"required,min=1,dive,uuid4"`
}
