package dms

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/anacrolix/ffprobe"
	"github.com/xbapps/xbvr/pkg/dms/dlna"
	"github.com/xbapps/xbvr/pkg/dms/upnp"
	"github.com/xbapps/xbvr/pkg/dms/upnpav"
	"gopkg.in/resty.v1"
)

type jsonInfo struct {
	DisplayName string `json:"display"`
}

type browse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int
	RequestedCount int
}

type contentDirectoryService struct {
	*Server
	upnp.Eventing
}

func FormatDurationSexagesimal(d time.Duration) string {
	ns := d % time.Second
	d /= time.Second
	s := d % 60
	d /= 60
	m := d % 60
	d /= 60
	h := d
	ret := fmt.Sprintf("%d:%02d:%02d.%09d", h, m, s, ns)
	ret = strings.TrimRight(ret, "0")
	ret = strings.TrimRight(ret, ".")
	return ret
}

func (cds *contentDirectoryService) updateIDString() string {
	return fmt.Sprintf("%d", uint32(os.Getpid()))
}

// Turns the given entry and DMS host into a UPnP object. A nil object is
// returned if the entry is not of interest.
func (me *contentDirectoryService) cdsObjectToUpnpavObject(cdsObject object, fileInfo os.FileInfo, host, userAgent string) (ret interface{}, err error) {
	log.Println("cdsObjectToUpnpavObject", cdsObject.ID())

	entryFilePath := cdsObject.FilePath()

	ignored, err := me.IgnorePath(entryFilePath)
	if err != nil {
		return
	}
	if ignored {
		return
	}
	obj := upnpav.Object{
		ID:         cdsObject.ID(),
		Restricted: 1,
		ParentID:   cdsObject.ParentID(),
	}
	if fileInfo.IsDir() {
		obj.Class = "object.container.storageFolder"
		obj.Title = fileInfo.Name()
		ret = upnpav.Container{Object: obj}
		return
	}

	// Check if JSON file is available
	jsonPath := cdsObject.FilePath() + ".json"
	if _, errt := os.Stat(jsonPath); errt == nil {
		b, errt := ioutil.ReadFile(jsonPath)
		if errt != nil {
			return
		}

		dat := &jsonInfo{}
		if errt := json.Unmarshal(b, &dat); errt != nil {
			return
		}

		obj.Title = dat.DisplayName + "_180_180x180_3dh_LR.mp4"
	}

	if !fileInfo.Mode().IsRegular() {
		// log.Printf("%s ignored: non-regular file", cdsObject.FilePath())
		return
	}
	mimeType, err := MimeTypeByPath(entryFilePath)
	if err != nil {
		return
	}
	// Use IsMedia() below if images should be shown
	if !mimeType.IsVideo() {
		// log.Printf("%s ignored: non-media file (%s)", cdsObject.FilePath(), mimeType)
		return
	}
	iconURI := (&url.URL{
		Scheme: "http",
		Host:   host,
		Path:   iconPath,
		RawQuery: url.Values{
			"path": {cdsObject.Path},
		}.Encode(),
	}).String()
	obj.Icon = iconURI
	// TODO(anacrolix): This might not be necessary due to item res image
	// element.
	obj.AlbumArtURI = iconURI
	obj.Class = "object.item." + mimeType.Type() + "Item"
	var (
		ffInfo        *ffprobe.Info
		nativeBitrate uint
		resDuration   string
	)
	if obj.Title == "" {
		obj.Title = fileInfo.Name()
	}
	resolution := func() string {
		if ffInfo != nil {
			for _, strm := range ffInfo.Streams {
				if strm["codec_type"] != "video" {
					continue
				}
				width := strm["width"]
				height := strm["height"]
				return fmt.Sprintf("%.0fx%.0f", width, height)
			}
		}
		return ""
	}()
	item := upnpav.Item{
		Object: obj,
		// Capacity: 1 for raw, 1 for icon, plus transcodes.
		Res: make([]upnpav.Resource, 0, 2+len(transcodes)),
	}
	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   resPath,
			RawQuery: url.Values{
				"path": {cdsObject.Path},
			}.Encode(),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Bitrate:    nativeBitrate,
		Duration:   resDuration,
		Size:       uint64(fileInfo.Size()),
		Resolution: resolution,
	})
	if mimeType.IsVideo() {
		if !me.NoTranscode {
			item.Res = append(item.Res, transcodeResources(host, cdsObject.Path, resolution, resDuration)...)
		}
	}
	if mimeType.IsVideo() || mimeType.IsImage() {
		item.Res = append(item.Res, upnpav.Resource{
			URL: (&url.URL{
				Scheme: "http",
				Host:   host,
				Path:   iconPath,
				RawQuery: url.Values{
					"path": {cdsObject.Path},
					"c":    {"jpeg"},
				}.Encode(),
			}).String(),
			ProtocolInfo: "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN",
		})
	}
	ret = item
	return
}

func (me *contentDirectoryService) xbaseFileToContainer(file XbaseFile, parent string, host string) interface{} {
	obj := upnpav.Object{
		ID:         fmt.Sprintf("file-%v", file.ID),
		Restricted: 1,
		ParentID:   parent,
		Title:      file.Filename,
	}

	mimeType := "video/mp4"

	item := upnpav.Item{
		Object: obj,
		Res:    make([]upnpav.Resource, 0, 2),
	}

	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   resPath,
			RawQuery: url.Values{
				"file": {fmt.Sprintf("%v", file.ID)},
			}.Encode(),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Bitrate: file.VideoBitrate,
		// Duration:   resDuration,
		Size: uint64(file.Size),
		// Resolution: resolution,
	})

	return item
}

func (me *contentDirectoryService) sceneToContainer(scene XbaseScene, parent string, host string) interface{} {
	c := make([]string, 0)
	for i := range scene.Cast {
		c = append(c, scene.Cast[i].Name)
	}

	if len(scene.File) == 0 {
		return nil
	}

	iconURI := (&url.URL{
		Scheme: "http",
		Host:   host,
		Path:   iconPath,
		RawQuery: url.Values{
			"scene": {scene.SceneID},
			"c":     {"jpeg"},
		}.Encode(),
	}).String()

	// Object goes first
	obj := upnpav.Object{
		ID:          scene.SceneID,
		Restricted:  1,
		ParentID:    parent,
		Title:       strings.Join(c, ", ") + " - " + scene.Title + " _180_180x180_3dh_LR.mp4",
		Icon:        iconURI,
		AlbumArtURI: iconURI,
	}

	// Wrap up
	item := upnpav.Item{
		Object: obj,
		Res:    make([]upnpav.Resource, 0, 2),
	}

	file := scene.File[0]
	mimeType := "video/mp4"

	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   resPath,
			RawQuery: url.Values{
				"scene": {scene.SceneID},
			}.Encode(),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Bitrate: file.VideoBitrate,
		// Duration:   resDuration,
		Size: uint64(file.Size),
		// Resolution: resolution,
	})

	item.Res = append(item.Res, upnpav.Resource{
		URL:          iconURI,
		ProtocolInfo: "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_MED",
	})

	return item
}

// Returns all the upnpav objects in a directory.
func (me *contentDirectoryService) readContainer(o object, host, userAgent string) (ret []interface{}, err error) {
	sfis := sortableFileInfoSlice{
		// TODO(anacrolix): Dig up why this special cast was added.
		FoldersLast: strings.Contains(userAgent, `AwoX/1.1`),
	}
	sfis.fileInfoSlice, err = o.readDir()
	if err != nil {
		return
	}
	sort.Sort(sfis)
	for _, fi := range sfis.fileInfoSlice {
		child := object{path.Join(o.Path, fi.Name()), me.RootObjectPath}
		obj, err := me.cdsObjectToUpnpavObject(child, fi, host, userAgent)
		if err != nil {
			log.Printf("error with %s: %s", child.FilePath(), err)
			continue
		}
		if obj != nil {
			ret = append(ret, obj)
		}
	}
	return
}

// ContentDirectory object from ObjectID.
func (me *contentDirectoryService) objectFromID(id string) (o object, err error) {
	o.Path, err = url.QueryUnescape(id)
	if err != nil {
		return
	}
	if o.Path == "0" {
		o.Path = "/"
	}
	// o.Path = path.Clean(o.Path)
	// if !path.IsAbs(o.Path) {
	// 	err = fmt.Errorf("bad ObjectID %v", o.Path)
	// 	return
	// }
	o.RootObjectPath = me.RootObjectPath

	return
}

func (me *contentDirectoryService) Handle(action string, argsXML []byte, r *http.Request) (map[string]string, error) {
	host := r.Host
	// userAgent := r.UserAgent()
	switch action {
	case "GetSystemUpdateID":
		return map[string]string{
			"Id": me.updateIDString(),
		}, nil
	case "GetSortCapabilities":
		return map[string]string{
			"SortCaps": "dc:title",
		}, nil
	case "Browse":
		var browse browse
		if err := xml.Unmarshal([]byte(argsXML), &browse); err != nil {
			return nil, err
		}

		obj, err := me.objectFromID(browse.ObjectID)
		if err != nil {
			return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, err.Error())
		}

		switch browse.BrowseFlag {
		case "BrowseDirectChildren":
			// Read folder and return children
			// TODO: check if obj == 0 and return root objects
			// TODO: check if special path and return files

			var objs []interface{}

			if obj.IsRoot() {
				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "actors",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "actors",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "tags",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "tags",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "released",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "released",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "sites",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "sites",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "watchlist",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "watchlist",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "favourites",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "favourites",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "all",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "all",
				}})

				objs = append(objs, upnpav.Container{Object: upnpav.Object{
					ID:         "not-matched",
					Restricted: 1,
					ParentID:   "0",
					Class:      "object.container.storageFolder",
					Title:      "not-matched",
				}})
			}

			// All videos
			if obj.Path == "all" {
				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "all", host))
				}
			}

			// Sites
			if obj.Path == "sites" {
				data := XbaseGet()

				for i := range data.Sites {
					objs = append(objs, upnpav.Container{Object: upnpav.Object{
						ID:         "sites/" + data.Sites[i],
						Restricted: 1,
						ParentID:   "sites",
						Class:      "object.container.storageFolder",
						Title:      data.Sites[i],
					}})
				}
			}

			if strings.HasPrefix(obj.Path, "sites/") {
				id := strings.Split(obj.Path, "/")

				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true, "sites":["` + id[1] + `"]}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "sites/"+id[1], host))
				}
			}

			// Tags
			if obj.Path == "tags" {
				data := XbaseGet()

				for i := range data.Tags {
					objs = append(objs, upnpav.Container{Object: upnpav.Object{
						ID:         "tags/" + data.Tags[i],
						Restricted: 1,
						ParentID:   "tags",
						Class:      "object.container.storageFolder",
						Title:      data.Tags[i],
					}})
				}
			}

			if strings.HasPrefix(obj.Path, "tags/") {
				id := strings.Split(obj.Path, "/")

				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true, "tags":["` + id[1] + `"]}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "tags/"+id[1], host))
				}
			}

			// Actors
			if obj.Path == "actors" {
				data := XbaseGet()

				for i := range data.Actors {
					objs = append(objs, upnpav.Container{Object: upnpav.Object{
						ID:         "actors/" + data.Actors[i],
						Restricted: 1,
						ParentID:   "actors",
						Class:      "object.container.storageFolder",
						Title:      data.Actors[i],
					}})
				}
			}

			if strings.HasPrefix(obj.Path, "actors/") {
				id := strings.Split(obj.Path, "/")

				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true, "cast":["` + id[1] + `"]}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "actors/"+id[1], host))
				}
			}

			// Release date
			if obj.Path == "released" {
				data := XbaseGet()

				for i := range data.ReleaseGroup {
					objs = append(objs, upnpav.Container{Object: upnpav.Object{
						ID:         "released/" + data.ReleaseGroup[i],
						Restricted: 1,
						ParentID:   "released",
						Class:      "object.container.storageFolder",
						Title:      data.ReleaseGroup[i],
					}})
				}
			}

			if strings.HasPrefix(obj.Path, "released/") {
				id := strings.Split(obj.Path, "/")

				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true, "releaseMonth":"` + id[1] + `"}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "released/"+id[1], host))
				}
			}

			// Watchlist
			if obj.Path == "watchlist" {
				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true, "lists":["watchlist"]}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "watchlist", host))
				}
			}

			// Favourites
			if obj.Path == "favourites" {
				var data XbaseScenes
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"isAccessible":true, "lists":["favourite"]}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/scene/list")

				for i := range data.Scenes {
					objs = append(objs, me.sceneToContainer(data.Scenes[i], "favourites", host))
				}
			}

			// Unmatched
			if obj.Path == "not-matched" {
				var data []XbaseFile
				resty.R().SetHeader("Content-Type", "application/json").
					SetBody(`{"state": "unmatched"}`).
					SetResult(&data).Post("http://127.0.0.1:9999/api/files/list")

				for i := range data {
					if _, err := os.Stat(filepath.Join(data[i].Path, data[i].Filename)); err == nil {
						objs = append(objs, me.xbaseFileToContainer(data[i], "unmatched", host))
					}
				}
			}

			result, err := xml.Marshal(objs)
			if err != nil {
				return nil, err
			}

			return map[string]string{
				"TotalMatches":   fmt.Sprint(len(objs)),
				"NumberReturned": fmt.Sprint(len(objs)),
				"Result":         didl_lite(string(result)),
				"UpdateID":       me.updateIDString(),
			}, nil
		// case "BrowseMetadata":
		// 	fileInfo, err := os.Stat(obj.FilePath())
		// 	if err != nil {
		// 		if os.IsNotExist(err) {
		// 			return nil, &upnp.Error{
		// 				Code: upnpav.NoSuchObjectErrorCode,
		// 				Desc: err.Error(),
		// 			}
		// 		}
		// 		return nil, err
		// 	}
		// 	upnp, err := me.cdsObjectToUpnpavObject(obj, fileInfo, host, userAgent)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	buf, err := xml.Marshal(upnp)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	return map[string]string{
		// 		"TotalMatches":   "1",
		// 		"NumberReturned": "1",
		// 		"Result":         didl_lite(func() string { return string(buf) }()),
		// 		"UpdateID":       me.updateIDString(),
		// 	}, nil
		default:
			return nil, upnp.Errorf(upnp.ArgumentValueInvalidErrorCode, "unhandled browse flag: %v", browse.BrowseFlag)
		}
	case "GetSearchCapabilities":
		return map[string]string{
			"SearchCaps": "",
		}, nil
	default:
		return nil, upnp.InvalidActionError
	}
}

// Returns the number of children this object has, such as for a container.
func (cds *contentDirectoryService) objectChildCount(me object) int {
	objs, err := cds.readContainer(me, "", "")
	if err != nil {
		log.Printf("error reading container: %s", err)
	}
	return len(objs)
}

func (cds *contentDirectoryService) objectHasChildren(obj object) bool {
	return cds.objectChildCount(obj) != 0
}

// Represents a ContentDirectory object.
type object struct {
	Path           string // The cleaned, absolute path for the object relative to the server.
	RootObjectPath string
}

// Returns the actual local filesystem path for the object.
func (o *object) FilePath() string {
	return filepath.Join(o.RootObjectPath, filepath.FromSlash(o.Path))
}

// Returns the ObjectID for the object. This is used in various ContentDirectory actions.
func (o object) ID() string {
	if !path.IsAbs(o.Path) {
		log.Panicf("Relative object path: %s", o.Path)
	}
	if len(o.Path) == 1 {
		return "0"
	}
	return url.QueryEscape(o.Path)
}

func (o *object) IsRoot() bool {
	return o.Path == "/"
}

// Returns the object's parent ObjectID. Fortunately it can be deduced from the
// ObjectID (for now).
func (o object) ParentID() string {
	if o.IsRoot() {
		return "-1"
	}
	o.Path = path.Dir(o.Path)
	return o.ID()
}

// This function exists rather than just calling os.(*File).Readdir because I
// want to stat(), not lstat() each entry.
func (o *object) readDir() (fis []os.FileInfo, err error) {
	dirPath := o.FilePath()
	dirFile, err := os.Open(dirPath)
	if err != nil {
		return
	}
	defer dirFile.Close()
	var dirContent []string
	dirContent, err = dirFile.Readdirnames(-1)
	if err != nil {
		return
	}
	fis = make([]os.FileInfo, 0, len(dirContent))
	for _, file := range dirContent {
		fi, err := os.Stat(filepath.Join(dirPath, file))
		if err != nil {
			continue
		}
		fis = append(fis, fi)
		// spew.Dump(fi)
	}
	return
}

type sortableFileInfoSlice struct {
	fileInfoSlice []os.FileInfo
	FoldersLast   bool
}

func (me sortableFileInfoSlice) Len() int {
	return len(me.fileInfoSlice)
}

func (me sortableFileInfoSlice) Less(i, j int) bool {
	if me.fileInfoSlice[i].IsDir() && !me.fileInfoSlice[j].IsDir() {
		return !me.FoldersLast
	}
	if !me.fileInfoSlice[i].IsDir() && me.fileInfoSlice[j].IsDir() {
		return me.FoldersLast
	}
	return strings.ToLower(me.fileInfoSlice[i].Name()) < strings.ToLower(me.fileInfoSlice[j].Name())
}

func (me sortableFileInfoSlice) Swap(i, j int) {
	me.fileInfoSlice[i], me.fileInfoSlice[j] = me.fileInfoSlice[j], me.fileInfoSlice[i]
}
