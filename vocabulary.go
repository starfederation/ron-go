package ron

// EnableVocabularies enables validation for supported typed vocabulary URIs.
// Unsupported typed values remain ordinary JSON/RON objects unless their vocabulary is enabled.
func EnableVocabularies(uris ...string) Option {
	return func(opts *optionState) {
		if opts.vocabularies == nil {
			opts.vocabularies = make(map[string]struct{}, len(uris))
		}
		for _, uri := range uris {
			opts.vocabularies[uri] = struct{}{}
		}
	}
}

func (opts optionState) hasVocabularies() bool {
	return len(opts.vocabularies) > 0
}

func (opts optionState) validateVocabularies(value any) error {
	_, err := opts.parseVocabularies(value)
	return err
}

func (opts optionState) parseVocabularies(value any) (any, error) {
	for uri := range opts.vocabularies {
		if !isSupportedVocabulary(uri) {
			return nil, newError("unsupported vocabulary: " + uri)
		}
	}
	return opts.parseVocabularyValue(value)
}

func isSupportedVocabulary(uri string) bool {
	switch uri {
	case VocabularyCoreV1:
		return true
	default:
		return false
	}
}

func (opts optionState) parseVocabularyValue(value any) (any, error) {
	switch value := value.(type) {
	case []any:
		for i, child := range value {
			parsed, err := opts.parseVocabularyValue(child)
			if err != nil {
				return nil, err
			}
			value[i] = parsed
		}
		return value, nil
	case map[string]any:
		members := objectMembers(value, false)
		if tag, payload, ok := opts.enabledTypedValue(members); ok {
			if len(members) != 1 {
				return nil, newError("typed vocabulary object must have exactly one member")
			}
			return opts.parseTypedPayload(tag, payload)
		}
		for key, child := range value {
			parsed, err := opts.parseVocabularyValue(child)
			if err != nil {
				return nil, err
			}
			value[key] = parsed
		}
		return value, nil
	case orderedObject:
		if tag, payload, ok := opts.enabledTypedValue(value.Members); ok {
			if len(value.Members) != 1 {
				return nil, newError("typed vocabulary object must have exactly one member")
			}
			return opts.parseTypedPayload(tag, payload)
		}
		for i, member := range value.Members {
			parsed, err := opts.parseVocabularyValue(member.Value)
			if err != nil {
				return nil, err
			}
			value.Members[i].Value = parsed
		}
		return value, nil
	default:
		return value, nil
	}
}

func (opts optionState) enabledTypedValue(members []objectMember) (string, any, bool) {
	for _, member := range members {
		if opts.isCoreTag(member.Key) {
			return member.Key, member.Value, true
		}
	}
	return "", nil, false
}

func (opts optionState) parseTypedPayload(tag string, payload any) (any, error) {
	if opts.isCoreTag(tag) {
		return opts.parseCorePayload(tag, payload)
	}
	return nil, newError("unsupported typed tag")
}
