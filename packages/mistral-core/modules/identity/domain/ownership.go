package domain

import "errors"

type Ownership struct {
	SubjectID   string `json:"subject_id"`
	CharacterID string `json:"character_id"`
}

func NewOwnership(subjectID, characterID string) (Ownership, error) {
	if subjectID == "" {
		return Ownership{}, errors.New("subject id is required")
	}
	if characterID == "" {
		return Ownership{}, errors.New("character id is required")
	}
	return Ownership{SubjectID: subjectID, CharacterID: characterID}, nil
}
