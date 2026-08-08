package main

// Fetches the maps of a Steam Workshop collection. Both endpoints are
// public and need no API key.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type WorkshopMap struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func FetchWorkshopMaps(collectionID string) ([]WorkshopMap, error) {
	if collectionID == "" {
		return nil, nil
	}
	ids, err := fetchCollectionChildren(collectionID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		// Not a collection — maybe the ID is a single workshop map.
		ids = []string{collectionID}
	}
	return fetchMapDetails(ids)
}

func fetchCollectionChildren(collectionID string) ([]string, error) {
	resp, err := http.PostForm(
		"https://api.steampowered.com/ISteamRemoteStorage/GetCollectionDetails/v1/?format=json",
		url.Values{"collectioncount": {"1"}, "publishedfileids[0]": {collectionID}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Response struct {
			CollectionDetails []struct {
				Children []struct {
					PublishedFileID string `json:"publishedfileid"`
				} `json:"children"`
			} `json:"collectiondetails"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var ids []string
	for _, c := range data.Response.CollectionDetails {
		for _, child := range c.Children {
			ids = append(ids, child.PublishedFileID)
		}
	}
	return ids, nil
}

func fetchMapDetails(ids []string) ([]WorkshopMap, error) {
	form := url.Values{"itemcount": {strconv.Itoa(len(ids))}}
	for i, id := range ids {
		form.Set(fmt.Sprintf("publishedfileids[%d]", i), id)
	}
	resp, err := http.PostForm(
		"https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/?format=json",
		form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Response struct {
			PublishedFileDetails []struct {
				PublishedFileID string `json:"publishedfileid"`
				Title           string `json:"title"`
				Result          int    `json:"result"`
			} `json:"publishedfiledetails"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var maps []WorkshopMap
	for _, d := range data.Response.PublishedFileDetails {
		if d.Result == 1 && d.Title != "" {
			maps = append(maps, WorkshopMap{ID: d.PublishedFileID, Title: d.Title})
		}
	}
	if len(maps) == 0 {
		return nil, errors.New("no workshop maps found for this collection ID")
	}
	return maps, nil
}
