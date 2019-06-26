package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

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
	User       string
	Pass       string
	UserId     string
	Token      string
	GraphqlURL string
}

type cardMsg struct {
	BoardId  string
	ListId   string
	CardId   string
	Title    string
	ParentId string
}

func main() {
	PORT, ok := os.LookupEnv("PORT")
	if !ok {
		PORT = "80"
	}
	HOST, ok := os.LookupEnv("HOST")
	if !ok {
		HOST = "0.0.0.0"
	}
	USER := os.Getenv("USER")
	PASS := os.Getenv("PASS")
	GRAPHQL_URL := os.Getenv("GRAPHQL_URL")

	cnf := config{
		User:       USER,
		Pass:       PASS,
		GraphqlURL: GRAPHQL_URL,
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
		if cnf.Token == "" {
			err := cnf.getToken()
			if err != nil {
				log.Printf("error in getToken: %v", err)
			}
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
		"act-archivedCard":
		card := cardMsg{
			BoardId: m.BoardId,
			ListId:  m.ListId,
			CardId:  m.CardId,
		}
		err := cnf.fillCard(&card)
		if err != nil {
			return fmt.Errorf("error in fillCard: %v", err)
		}
		parent, err := cnf.findParent(card)
		if err != nil {
			return fmt.Errorf("error in findParent: %v", err)
		}
		if parent == nil {
			return nil
		}
		switch m.Description {
		case "act-createCard":
			err = cnf.setCheckListItem(*parent, card.Title, "Pronto", false)
			if err != nil {
				return fmt.Errorf("error in setCheckListItem: %v", err)
			}
		case "act-archivedCard":
			err = cnf.setCheckListItem(*parent, card.Title, "Pronto", true)
			if err != nil {
				return fmt.Errorf("error in setCheckListItem: %v", err)
			}
		}
	}
	return nil
}

func (cnf *config) findParent(c cardMsg) (*cardMsg, error) {
	if c.ParentId == "" {
		return nil, nil
	}
	cards, err := cnf.allCards()
	if err != nil {
		return nil, err
	}
	for _, x := range cards {
		if x.CardId == c.ParentId {
			return &x, nil
		}
	}
	return nil, nil
}

func (cnf *config) setCheckListItem(c cardMsg, checkListTitle string, itemTitle string, isFinished bool) error {
	q := fmt.Sprintf(`
	mutation{
		setCheckListItem(
		  auth:{
			userId: "%s",
			  token: "%s"
		  }
		  boardId:"%s"
		  cardId:"%s"
		  checkListTitle: "%s"
		  itemTitle: "%s"
		  isFinished: %v
		)
	  }
	`, cnf.UserId, cnf.Token, c.BoardId, c.CardId, checkListTitle, itemTitle, isFinished)
	d := struct {
		Errors []struct {
			Message string
		}
	}{}
	err := cnf.query(q, &d)
	if err != nil {
		return err
	}
	if len(d.Errors) > 0 {
		return fmt.Errorf(d.Errors[0].Message)
	}
	return nil
}

func (cnf *config) allCards() ([]cardMsg, error) {
	q := fmt.Sprintf(`
	query{
		boards(auth: {userId: "%s", token: "%s"}){
			_id
			lists{
				_id
				cards{
					_id
				}
			}
		}
	}
	`, cnf.UserId, cnf.Token)
	d := struct {
		Errors []struct {
			Message string
		}
		Data struct {
			Boards []struct {
				ID    string `json:"_id"`
				Lists []struct {
					ID    string `json:"_id"`
					Cards []struct {
						ID string `json:"_id"`
					}
				}
			}
		}
	}{}
	cards := []cardMsg{}
	err := cnf.query(q, &d)
	if err != nil {
		return cards, err
	}
	if len(d.Errors) > 0 {
		return cards, fmt.Errorf(d.Errors[0].Message)
	}
	for _, b := range d.Data.Boards {
		for _, l := range b.Lists {
			for _, cd := range l.Cards {
				cards = append(cards, cardMsg{
					BoardId: b.ID,
					ListId:  l.ID,
					CardId:  cd.ID,
				})
			}
		}
	}
	return cards, nil
}

func (cnf *config) getToken() error {
	q := fmt.Sprintf(`
	query{
		authorize(user:"%s", password:"%s"){
			userId
			token
		}
	}
	`, cnf.User, cnf.Pass)
	d := struct {
		Errors []struct {
			Message string
		}
		Data struct {
			Authorize struct {
				UserId string
				Token  string
			}
		}
	}{}
	err := cnf.query(q, &d)
	if err != nil {
		return err
	}
	if len(d.Errors) > 0 {
		return fmt.Errorf(d.Errors[0].Message)
	}
	cnf.UserId = d.Data.Authorize.UserId
	cnf.Token = d.Data.Authorize.Token
	return nil
}

func (cnf *config) fillCard(card *cardMsg) error {
	q := fmt.Sprintf(`
	query{
		board(
			auth: {userId: "%s", token: "%s"}
			_id: "%s"
		){
			list(_id: "%s"){
				card(_id: "%s"){
					title
					parentId
				}
			}
		}
	}
	`, cnf.UserId, cnf.Token, card.BoardId, card.ListId, card.CardId)
	d := struct {
		Errors []struct {
			Message string
		}
		Data struct {
			Board struct {
				List struct {
					Card struct {
						Title    string
						ParentId string
					}
				}
			}
		}
	}{}
	err := cnf.query(q, &d)
	if err != nil {
		return fmt.Errorf("error in query: %v", err)
	}
	if len(d.Errors) > 0 {
		return fmt.Errorf("graphql error: %v", d.Errors[0].Message)
	}
	card.ParentId = d.Data.Board.List.Card.ParentId
	card.Title = d.Data.Board.List.Card.Title
	return nil
}

func (cnf *config) query(q string, v interface{}) error {
	// log.Println(q)
	r, err := http.Post(cnf.GraphqlURL, "application/graphql", strings.NewReader(q))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, v)
	if err != nil {
		return err
	}
	return nil
}
