package hooks

type CardMsg struct {
	ID           string `bson:"_id"`
	Title        string `bson:"title"`
	ParentID     string `bson:"parentId"`
	BoardID      string `bson:"boardId"`
	CustomFields []struct {
		ID    string      `bson:"_id"`
		Value interface{} `bson:"value"`
	} `bson:"customFields"`
}

type Checklist struct {
	ID     string `bson:"_id"`
	Title  string `bson:"title"`
	CardID string `bson:"cardId"`
}

// Hooker receives an act and trigger some reaction
type Hooker func(act string, cardId string, ops Operations) error

type Operations interface {
	SetCheckListItem(cardId string, checkListTitle string, itemTitle string, isFinished bool) error
	FindCard(cardId string) (CardMsg, error)
	FindBoard(title string) (id string, ok bool, err error)
	FindCustomField(title, boardId string) (id string, ok bool, err error)
	SetCustomField(cardID, fieldID, value string) error
}

const ActAddBoardMember = "act-addBoardMember"
const ActAddChecklist = "act-addChecklist"
const ActAddChecklistItem = "act-addChecklistItem"
const ActAddedLabel = "act-addedLabel"
const ActArchivedCard = "act-archivedCard"
const ActArchivedList = "act-archivedList"
const ActArchivedSwimlane = "act-archivedSwimlane"
const ActCheckedItem = "act-checkedItem"
const ActCompleteChecklist = "act-completeChecklist"
const ActCreateCard = "act-createCard"
const ActCreateCustomField = "act-createCustomField"
const ActCreateList = "act-createList"
const ActCreateSwimlane = "act-createSwimlane"
const ActJoinMember = "act-joinMember"
const ActMoveCard = "act-moveCard"
const ActRemoveChecklist = "act-removeChecklist"
const ActRemovedChecklistItem = "act-removedChecklistItem"
const ActRemovedLabel = "act-removedLabel"
const ActRestoredCard = "act-restoredCard"
const ActSetCustomField = "act-setCustomField"
const ActUncheckedItem = "act-uncheckedItem"
const ActUncompleteChecklist = "act-uncompleteChecklist"
const ActUnjoinMember = "act-unjoinMember"
const ActUnsetCustomField = "act-unsetCustomField"
