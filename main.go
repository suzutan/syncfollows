package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	t "github.com/suzutan/syncfollowlist/internal/pkg/twitter"
)

type contextKey string

const contextClient contextKey = "client"
const contextListID contextKey = "listID"

func do(ctx context.Context) {

	client := ctx.Value(contextClient).(*twitter.Client)
	listID := ctx.Value(contextListID).(int64)

	// get follows
	log.Print("fetch friend IDs")
	friendIDs, _, err := client.Friends.IDs(&twitter.FriendIDParams{
		Count: 5000,
	})
	if err != nil {
		log.Print(err)
		return
	}

	// get follows list members
	log.Print("fetch List IDs")

	listMembers, _, err := client.Lists.Members(&twitter.ListsMembersParams{
		ListID: listID,
		Count:  5000,
	})
	if err != nil {
		log.Print(err)
		return
	}

	// map list members to IDs
	var listIDs []int64
	for _, member := range listMembers.Users {
		listIDs = append(listIDs, member.ID)
	}

	var addIDs = Int64ListDivide(friendIDs.IDs, listIDs)
	var delIDs = Int64ListDivide(listIDs, friendIDs.IDs)

	//  add follows to list
	if len(addIDs) > 0 {
		res, err := client.Lists.MembersCreateAll(&twitter.ListsMembersCreateAllParams{
			ListID: listID,
			UserID: strings.Trim(strings.Join(strings.Fields(fmt.Sprint(addIDs)), ","), "[]"),
		})

		if err != nil {
			log.Print(err)
			return
		}
		if res.StatusCode == http.StatusOK {
			log.Printf("add success. count:%d\n", len(addIDs))
		} else {
			log.Printf("add failed. %s", res.Status)
		}
	} else {
		log.Print("addIds is 0, skip.")
	}

	// remove follows from list
	if len(delIDs) > 0 {
		res, err := client.Lists.MembersDestroyAll(&twitter.ListsMembersDestroyAllParams{
			ListID: listID,
			UserID: strings.Trim(strings.Join(strings.Fields(fmt.Sprint(delIDs)), ","), "[]"),
		})

		if err != nil {
			log.Print(err)
			return
		}
		if res.StatusCode == http.StatusOK {
			log.Printf("delete success. count:%d\n", len(delIDs))
		} else {
			log.Printf("delete failed. %s", res.Status)
		}
	} else {
		log.Print("delIDs is 0, skip.")
	}
}

func run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	do(ctx)
	log.Printf("wait for %s\n", interval.String())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			do(ctx)
			log.Printf("wait for %s\n", interval.String())
		}
	}
}

func main() {

	auth := &t.AuthConfig{
		ConsumerKey:       os.Getenv("CK"),
		ConsumerSecret:    os.Getenv("CS"),
		AccessToken:       os.Getenv("AT"),
		AccessTokenSecret: os.Getenv("ATS"),
	}
	listID, _ := strconv.ParseInt(os.Getenv("LIST_ID"), 10, 64)
	client := t.New(auth)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = context.WithValue(ctx, contextClient, client)
	ctx = context.WithValue(ctx, contextListID, listID)

	run(ctx, 1*time.Minute)

}

// Int64ListDivide mainListを総当りし、divideListに存在しないrecordを抽出する
// [1,2,3] - [1,2] = [3]
// for i in [1,2,3]:
// 1: 1 in [1,2] -> true
// 2: 2 in [1,2] -> true
// 3: 3 in [1,2] -> false -> add 3 to sublist
// return [3]
func Int64ListDivide(mainList []int64, divideList []int64) []int64 {
	var result []int64
	for _, id := range mainList {
		if !int64Contains(divideList, id) {
			result = append(result, id)
		}
	}
	return result
}

func int64Contains(list []int64, target int64) bool {
	for _, id := range list {
		if id == target {
			return true
		}
	}
	return false
}