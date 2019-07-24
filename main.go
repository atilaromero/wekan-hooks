package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/setecrs/wekan-hooks/hooks/fields"

	"github.com/pkg/errors"

	"github.com/setecrs/wekan-hooks/hooks"
	"github.com/setecrs/wekan-hooks/hooks/child"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func uuid() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var b bytes.Buffer
	for i := 0; i < 17; i++ {
		b.WriteByte(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

type hookMsg struct {
	Text        string
	CardId      string
	ListId      string
	BoardId     string
	User        string
	Card        string
	SwimlaneId  string
	Description string
}

type config struct {
	MongoClient *mongo.Client
	Hooks       []hooks.Hooker
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	PORT, ok := os.LookupEnv("PORT")
	if !ok {
		PORT = "80"
	}
	HOST, ok := os.LookupEnv("HOST")
	if !ok {
		HOST = "0.0.0.0"
	}
	MONGO_URL, ok := os.LookupEnv("MONGO_URL")
	if !ok {
		panic("MONGO_URL not set. Example: mongodb://localhost:27017")
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(MONGO_URL))
	if err != nil {
		panic(err)
	}

	cnf := config{
		MongoClient: client,
		Hooks: []hooks.Hooker{
			child.Creation,
			child.Archive,
			fields.IPL,
			fields.Path,
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("error in ReadAll: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		data := hookMsg{}
		err = json.Unmarshal(buf, &data)
		if err != nil {
			log.Printf("error in Unmarshal: %v", err)
			return
		}
		err = cnf.processMsg(data)
		if err != nil {
			log.Printf("error in processMsg: %v", err)
			return
		}
	})
	http.ListenAndServe(fmt.Sprintf("%s:%s", HOST, PORT), nil)
}

func (cnf *config) processMsg(m hookMsg) error {
	log.Printf("%+v\n", m)
	switch m.Description {
	case "act-createCard",
		"act-archivedCard",
		hooks.ActMoveCard:
		for _, h := range cnf.Hooks {
			err := h(m.Description, m.CardId, cnf)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cnf *config) findChecklist(cardID, checklistTitle string) (id string, ok bool, err error) {
	coll := cnf.MongoClient.Database("wekan").Collection("checklists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := coll.FindOne(ctx, bson.M{"cardId": cardID, "title": checklistTitle})
	idStruct := struct {
		ID string `bson:"_id"`
	}{}
	err = result.Decode(&idStruct)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", false, nil
		}
		return "", false, err
	}
	return idStruct.ID, true, nil
}

func (cnf *config) insertChecklist(cardID, checklistTitle string) (id string, err error) {
	coll := cnf.MongoClient.Database("wekan").Collection("checklists")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id = uuid()
	_, err = coll.InsertOne(ctx, bson.M{"_id": id, "cardId": cardID, "title": checklistTitle})
	if err != nil {
		if err != nil {
			return "", errors.Wrap(err, "error inserting new checklist")
		}
	}
	return id, nil
}

func (cnf *config) findChecklistItem(cardID, checklistID, itemTitle string) (id string, ok bool, err error) {
	coll := cnf.MongoClient.Database("wekan").Collection("checklistItems")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := coll.FindOne(ctx, bson.M{"cardId": cardID, "checklistId": checklistID, "title": itemTitle})
	idStruct := struct {
		ID string `bson:"_id"`
	}{}
	err = result.Decode(&idStruct)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", false, nil
		}
		return "", false, err
	}
	return idStruct.ID, true, nil
}

func (cnf *config) insertChecklistItem(cardID, checklistID, itemTitle string, isFinished bool) (id string, err error) {
	coll := cnf.MongoClient.Database("wekan").Collection("checklistItems")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id = uuid()
	_, err = coll.InsertOne(ctx, bson.M{"_id": id, "cardId": cardID, "checklistId": checklistID, "title": itemTitle, "isFinished": isFinished})
	if err != nil {
		if err != nil {
			return "", errors.Wrap(err, "error inserting new checklistItem")
		}
	}
	return id, nil
}

func (cnf *config) updateChecklistItem(id string, isFinished bool) error {
	coll := cnf.MongoClient.Database("wekan").Collection("checklistItems")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"isFinished": isFinished}})
	if err != nil {
		if err != nil {
			return errors.Wrap(err, "error updating checklistItem")
		}
	}
	return nil
}

func (cnf config) SetCheckListItem(cardID string, checklistTitle string, itemTitle string, isFinished bool) error {
	chklstID, ok, err := cnf.findChecklist(cardID, checklistTitle)
	if err != nil {
		return errors.Wrap(err, "error finding checklist")
	}
	if !ok {
		chklstID, err = cnf.insertChecklist(cardID, checklistTitle)
	}
	itemID, ok, err := cnf.findChecklistItem(cardID, chklstID, itemTitle)
	if err != nil {
		return errors.Wrap(err, "error finding checklistItem")
	}
	if !ok {
		itemID, err = cnf.insertChecklistItem(cardID, chklstID, itemTitle, isFinished)
		return errors.Wrap(err, "error inserting checklistItem")
	}
	err = cnf.updateChecklistItem(itemID, isFinished)
	if err != nil {
		return errors.Wrap(err, "error updating checklistItem")
	}
	return nil
}

func (cnf config) FindCard(cardID string) (hooks.CardMsg, error) {
	coll := cnf.MongoClient.Database("wekan").Collection("cards")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := coll.FindOne(ctx, bson.M{"_id": cardID})
	card := hooks.CardMsg{}
	err := result.Decode(&card)
	return card, err
}

func (cnf config) FindBoard(title string) (id string, ok bool, err error) {
	coll := cnf.MongoClient.Database("wekan").Collection("boards")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := coll.FindOne(ctx, bson.M{"title": title})
	idStruct := struct {
		ID string `bson:"_id"`
	}{}
	err = result.Decode(&idStruct)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", false, nil
		}
		return "", false, err
	}
	return idStruct.ID, true, nil
}

func (cnf config) FindCustomField(name, boardID string) (id string, ok bool, err error) {
	coll := cnf.MongoClient.Database("wekan").Collection("customFields")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := coll.FindOne(ctx, bson.M{"name": name, "boardIds": boardID})
	idStruct := struct {
		ID string `bson:"_id"`
	}{}
	err = result.Decode(&idStruct)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", false, nil
		}
		return "", false, err
	}
	return idStruct.ID, true, nil
}

func (cnf config) SetCustomField(cardID, fieldID, value string) error {
	card, err := cnf.FindCard(cardID)
	if err != nil {
		return err
	}

	coll := cnf.MongoClient.Database("wekan").Collection("cards")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i, k := range card.CustomFields {
		if k.ID == fieldID {
			key := fmt.Sprintf("customFields.%d.value", i)
			fmt.Println("found key", key)
			_, err := coll.UpdateOne(
				ctx,
				bson.M{"_id": cardID},
				bson.M{"$set": bson.M{key: value}},
			)
			return err
		}
	}
	_, err = coll.UpdateOne(
		ctx,
		bson.M{"_id": cardID},
		bson.M{"$push": bson.M{"customFields": bson.M{"_id": fieldID, "value": value}}},
	)
	return err
}
