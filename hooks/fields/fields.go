package fields

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	hooks "github.com/setecrs/wekan-hooks/hooks"
)

type CustomFieldsIDs struct {
	ipl         string
	registro    string
	solicitacao string
	auto        string
	item        string
	erro        string
	path        string
}

var BoardMateriaisID string
var customFieldsIDs CustomFieldsIDs

func IPL(act string, cardId string, ops hooks.Operations) error {
	if act != hooks.ActMoveCard {
		return nil
	}
	err := checkIDs(ops)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("IPL: error checking IDs of custom fields"))
	}
	card, err := ops.FindCard(cardId)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not find card: %s", cardId))
	}
	if card.BoardID != BoardMateriaisID {
		return nil
	}
	for _, cf := range card.CustomFields {
		if cf.ID == customFieldsIDs.ipl {
			if cf.Value != nil {
				if fmt.Sprintf("%v", cf.Value) != "" {
					// ipl already filled
					return nil
				}
			}
		}
	}
	if card.ParentID == "" {
		// Material has no parent
		return nil
	}
	reg, err := ops.FindCard(card.ParentID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not find parent card: %s", card.ParentID))
	}
	if reg.ParentID == "" {
		// Material has no grand parent
		return nil
	}
	ipl, err := ops.FindCard(reg.ParentID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not find parent card: %s", card.ParentID))
	}
	err = ops.SetCustomField(cardId, customFieldsIDs.ipl, ipl.Title)
	if err != nil {
		return errors.Wrap(err, "could not update custom field ipl")
	}
	return nil
}

func Path(act string, cardId string, ops hooks.Operations) error {
	if act != hooks.ActMoveCard {
		return nil
	}
	err := checkIDs(ops)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Path: error checking IDs of custom fields"))
	}
	card, err := ops.FindCard(cardId)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not find card: %s", cardId))
	}
	if card.BoardID != BoardMateriaisID {
		return nil
	}
	for _, cf := range card.CustomFields {
		if cf.ID == customFieldsIDs.path {
			if cf.Value != nil {
				if fmt.Sprintf("%v", cf.Value) != "" {
					// path already filled
					return nil
				}
			}
		}
	}
	path, err := buildPath(card)
	if err != nil {
		return errors.Wrap(err, "could not buildPath")
	}
	err = ops.SetCustomField(cardId, customFieldsIDs.path, path)
	if err != nil {
		return errors.Wrap(err, "could not update custom field path")
	}
	return nil
}

func buildPath(card hooks.CardMsg) (string, error) {
	idValue := make(map[string]string)
	for _, cf := range card.CustomFields {
		if cf.Value == nil {
			continue
		}
		idValue[cf.ID] = fmt.Sprintf("%v", cf.Value)
	}
	var b bytes.Buffer
	b.WriteString("/operacoes/")
	if s, ok := idValue[customFieldsIDs.ipl]; ok {
		s = normalizeString(s)
		b.WriteString(s)
		b.WriteString("/")
	} else if s, ok := idValue[customFieldsIDs.registro]; ok {
		s = normalizeString(s)
		b.WriteString(s)
		b.WriteString("/")
	} else {
		return "", fmt.Errorf("card %s does not have ipl or registro in custom fields", card.ID)
	}
	if s, ok := idValue[customFieldsIDs.auto]; ok {
		s = normalizeString(s)
		b.WriteString("auto_apreensao_")
		b.WriteString(s)
		b.WriteString("/")
	}
	if s, ok := idValue[customFieldsIDs.item]; ok {
		s = normalizeString(s)
		b.WriteString("item")
		b.WriteString(s)
		b.WriteString("_")
		b.WriteString(normalizeString(card.Title))
		b.WriteString("/")
		b.WriteString("item")
		b.WriteString(s)
		b.WriteString("_")
	}
	b.WriteString(normalizeString(card.Title))
	b.WriteString(".dd")
	return b.String(), nil
}

func normalizeString(s string) string {
	s = strings.Split(s, "/")[0]
	reg, err := regexp.Compile("[^a-zA-Z0-9_-]+")
	if err != nil {
		log.Fatal(err)
	}
	s = reg.ReplaceAllString(s, "")
	return s
}

func checkIDs(ops hooks.Operations) error {
	if BoardMateriaisID == "" {
		id, ok, err := ops.FindBoard("Materiais")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error searching board"))
		}
		if !ok {
			return fmt.Errorf("Board Materiais not found.")
		}
		BoardMateriaisID = id
	}
	if customFieldsIDs.ipl == "" {
		id, err := getID(ops, "ipl")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.ipl = id
	}
	if customFieldsIDs.registro == "" {
		id, err := getID(ops, "registro")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.registro = id
	}
	if customFieldsIDs.solicitacao == "" {
		id, err := getID(ops, "solicitacao")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.solicitacao = id
	}
	if customFieldsIDs.auto == "" {
		id, err := getID(ops, "auto")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.auto = id
	}
	if customFieldsIDs.item == "" {
		id, err := getID(ops, "item")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.item = id
	}
	if customFieldsIDs.erro == "" {
		id, err := getID(ops, "erro")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.erro = id
	}
	if customFieldsIDs.path == "" {
		id, err := getID(ops, "path")
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("checkIDs: error getting ID"))
		}
		customFieldsIDs.path = id
	}
	return nil
}

func getID(ops hooks.Operations, title string) (string, error) {
	id, ok, err := ops.FindCustomField(title, BoardMateriaisID)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("getID: error searching for '%s'", title))
	}
	if !ok {
		return "", fmt.Errorf("custom field not found: %s", title)
	}
	return id, nil
}
