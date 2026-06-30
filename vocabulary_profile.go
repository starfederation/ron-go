package ron

import (
	"bytes"
	"encoding/json"
)

// VocabularyProfile declares required and optional typed vocabularies.
type VocabularyProfile struct {
	Vocabularies map[string]bool `json:"vocabularies"`
}

// ValidateVocabularyProfile rejects required vocabularies that ron-go does not support.
func ValidateVocabularyProfile(src []byte, options ...Option) error {
	opts := optionState{
		vocabularyMask: defaultVocabularySet,
	}
	for _, option := range options {
		option(&opts)
	}

	dec := json.NewDecoder(bytes.NewReader(src))
	dec.DisallowUnknownFields()
	var profile VocabularyProfile
	if err := dec.Decode(&profile); err != nil {
		return newError("invalid vocabulary profile")
	}
	if len(profile.Vocabularies) == 0 {
		return nil
	}
	for uri, required := range profile.Vocabularies {
		if required && !opts.supportsVocabulary(uri) {
			return newError("unsupported vocabulary: " + uri)
		}
	}
	return nil
}

func (opts optionState) supportsVocabulary(uri string) bool {
	switch uri {
	case VocabularyCoreV1, VocabularyTimeV1, VocabularyNetworkV1, VocabularyMathV1, VocabularySpatialV1, VocabularyGeoV1, VocabularyColorV1, VocabularySetV1:
		return true
	default:
		return opts.isCustomVocabulary(uri)
	}
}
