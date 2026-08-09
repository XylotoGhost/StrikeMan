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
	"strings"
)

type WorkshopMap struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"` // Steam category tags, e.g. "Classic", "Wingman"
}

func (m WorkshopMap) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// CheckServerUpToDate asks Steam whether a CS2 server build is current.
// The build number is the one `status` reports as "1.41.7.4/14174".
func CheckServerUpToDate(build int) (upToDate bool, latest int, note string, err error) {
	resp, err := http.Get(fmt.Sprintf(
		"https://api.steampowered.com/ISteamApps/UpToDateCheck/v1/?appid=730&version=%d", build))
	if err != nil {
		return false, 0, "", err
	}
	defer resp.Body.Close()

	var data struct {
		Response struct {
			Success         bool   `json:"success"`
			UpToDate        bool   `json:"up_to_date"`
			RequiredVersion int    `json:"required_version"`
			Message         string `json:"message"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, 0, "", err
	}
	if !data.Response.Success {
		return false, 0, "", errors.New("Steam could not check this build")
	}
	return data.Response.UpToDate, data.Response.RequiredVersion, data.Response.Message, nil
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
				Tags            []struct {
					Tag string `json:"tag"`
				} `json:"tags"`
			} `json:"publishedfiledetails"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var maps []WorkshopMap
	for _, d := range data.Response.PublishedFileDetails {
		if d.Result == 1 && d.Title != "" {
			m := WorkshopMap{ID: d.PublishedFileID, Title: d.Title, Tags: []string{}}
			for _, t := range d.Tags {
				m.Tags = append(m.Tags, t.Tag)
			}
			maps = append(maps, m)
		}
	}
	if len(maps) == 0 {
		return nil, errors.New("no workshop maps found for this collection ID")
	}
	return maps, nil
}
