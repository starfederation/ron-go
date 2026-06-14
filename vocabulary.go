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
	for uri := range opts.vocabularies {
		if !isSupportedVocabulary(uri) {
			return newError("unsupported vocabulary: " + uri)
		}
	}
	return opts.validateVocabularyValue(value)
}

func isSupportedVocabulary(uri string) bool {
	switch uri {
	case VocabularyCoreV1:
		return true
	default:
		return false
	}
}

func (opts optionState) validateVocabularyValue(value any) error {
	switch value := value.(type) {
	case []any:
		for _, child := range value {
			if err := opts.validateVocabularyValue(child); err != nil {
				return err
			}
		}
		return nil
	case map[string]any, orderedObject:
		members := objectMembers(value, false)
		if tag, payload, ok := opts.enabledTypedValue(members); ok {
			if len(members) != 1 {
				return newError("typed vocabulary object must have exactly one member")
			}
			return opts.validateTypedPayload(tag, payload)
		}
		for _, member := range members {
			if err := opts.validateVocabularyValue(member.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (opts optionState) enabledTypedValue(members []objectMember) (string, any, bool) {
	for _, member := range members {
		if opts.isCoreTag(member.Key) {
			return member.Key, member.Value, true
		}
	}
	return "", nil, false
}

func (opts optionState) validateTypedPayload(tag string, payload any) error {
	if opts.isCoreTag(tag) {
		return opts.validateCorePayload(tag, payload)
	}
	return newError("unsupported typed tag")
}
