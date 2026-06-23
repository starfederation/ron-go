package ron

import "strings"

// CustomParseFunc validates and maps a custom typed payload.
type CustomParseFunc func(tag string, payload any) (any, error)

// CustomRenderFunc maps a native value to a custom typed tag and payload.
type CustomRenderFunc func(value any) (tag string, payload any, ok bool)

// CustomVocabulary defines an option-scoped typed vocabulary.
type CustomVocabulary struct {
	URI    string
	Tags   []string
	Parse  CustomParseFunc
	Render CustomRenderFunc
}

// CustomValue is a generic custom typed value.
type CustomValue struct {
	Tag     string
	Payload any
}

// Custom returns a generic custom typed value.
func Custom(tag string, payload any) CustomValue {
	return CustomValue{
		Tag:     normalizeCustomTag(tag),
		Payload: payload,
	}
}

// UseCustomVocabulary enables an option-scoped custom vocabulary.
func UseCustomVocabulary(vocabulary CustomVocabulary) Option {
	return func(opts *optionState) {
		if opts.vocabularies == nil {
			opts.vocabularies = make(map[string]struct{}, 1)
		}
		if opts.customVocabularies == nil {
			opts.customVocabularies = make(map[string]CustomVocabulary, 1)
		}
		if opts.customTags == nil {
			opts.customTags = make(map[string]string, len(vocabulary.Tags))
		}
		if vocabulary.URI != "" {
			opts.vocabularies[vocabulary.URI] = struct{}{}
			if _, ok := opts.customVocabularies[vocabulary.URI]; !ok {
				opts.customVocabularyOrder = append(opts.customVocabularyOrder, vocabulary.URI)
			}
			opts.customVocabularies[vocabulary.URI] = vocabulary
		}
		for _, tag := range vocabulary.Tags {
			opts.customTags[normalizeCustomTag(tag)] = vocabulary.URI
		}
	}
}

func (opts optionState) isCustomVocabulary(uri string) bool {
	_, ok := opts.customVocabularies[uri]
	return ok
}

func (opts optionState) isCustomTag(tag string) bool {
	_, ok := opts.customTags[tag]
	return ok
}

func (opts optionState) parseCustomPayload(tag string, payload any) (any, error) {
	uri, ok := opts.customTags[tag]
	if !ok {
		return nil, newError("unsupported custom tag")
	}
	vocabulary, ok := opts.customVocabularies[uri]
	if !ok || vocabulary.Parse == nil {
		return Custom(tag, payload), nil
	}
	return vocabulary.Parse(tag, payload)
}

func (opts optionState) customRenderersList() []CustomRenderFunc {
	if len(opts.customVocabularies) == 0 {
		return nil
	}
	renderers := make([]CustomRenderFunc, 0, len(opts.customVocabularies))
	for _, uri := range opts.customVocabularyOrder {
		vocabulary := opts.customVocabularies[uri]
		if vocabulary.Render != nil {
			renderers = append(renderers, vocabulary.Render)
		}
	}
	return renderers
}

func normalizeCustomTag(tag string) string {
	if strings.HasPrefix(tag, "#") {
		return tag
	}
	return "#" + tag
}
