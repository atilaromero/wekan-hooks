package child

import (
	hooks "github.com/setecrs/wekan-hooks/hooks"
)

func Creation(act string, cardId string, ops hooks.Operations) error {
	if act != hooks.ActCreateCard {
		return nil
	}
	card, err := ops.FindCard(cardId)
	if err != nil {
		return err
	}
	if card.ParentID == "" {
		return nil
	}
	return ops.SetCheckListItem(card.ParentID, card.Title, "Pronto", false)
}

func Archive(act string, cardId string, ops hooks.Operations) error {
	if act != hooks.ActArchivedCard {
		return nil
	}
	card, err := ops.FindCard(cardId)
	if err != nil {
		return err
	}
	if card.ParentID == "" {
		return nil
	}
	return ops.SetCheckListItem(card.ParentID, card.Title, "Pronto", true)
}
