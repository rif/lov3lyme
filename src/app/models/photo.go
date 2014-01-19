package models

import (
	"fmt"
	"github.com/rif/forms"
	"github.com/rif/resize"
	"image"
	"image/draw"
	"image/jpeg"
	"labix.org/v2/mgo/bson"
	"os"
	"path"
	"time"
)

const (
	IMAGE_HEIGHT     = 800
	THUMB_HEIGHT     = 80
	PROCESSING_IMAGE = "processing"
	ERROR_IMAGE      = "error"
)

type Photo struct {
	Id                        bson.ObjectId `bson:"_id,omitempty"`
	Title                     string
	Description               string
	Active                    bool
	Deleted                   bool
	User                      bson.ObjectId
	Country, Location, Gender string
	Age                       int
	FakeCount                 int
	FakeReporters             []bson.ObjectId
	AbuseCount                int
	AbuseReporters            []bson.ObjectId
	UpdatedOn                 time.Time
	Comments                  []*Comment
	Rand                      int64
}

func (p *Photo) SaveImage(img image.Image, x1, y1, x2, y2 int) error {
	if x2 > 0 && y2 > 0 {
		if img.Bounds().Dy() > 440 {
			scaleFactor := float64(img.Bounds().Dy()) / 440.0
			x1 = int(float64(x1) * scaleFactor)
			y1 = int(float64(y1) * scaleFactor)
			x2 = int(float64(x2) * scaleFactor)
			y2 = int(float64(y2) * scaleFactor)
		}

		buf := image.NewRGBA(image.Rect(0, 0, x2-x1, y2-y1))
		draw.Draw(buf, buf.Bounds(), img, image.Pt(x1, y1), draw.Src)
		img = buf
	}
	fn := fmt.Sprintf("%s.jpg", p.Id.Hex())
	img, err := p.resize(img, fn, IMAGE_HEIGHT, IMAGE_HEIGHT)
	if err != nil {
		Log("resize err: ", err.Error())
		return err
	}
	fn = fmt.Sprintf("%s_thumb.jpg", p.Id.Hex())
	_, err = p.resize(img, fn, THUMB_HEIGHT, THUMB_HEIGHT)
	return err
}

func (p *Photo) resize(img image.Image, fn string, w, h int) (image.Image, error) {
	bounds := img.Bounds()
	if bounds.Dx() >= bounds.Dy() {
		h = bounds.Dy() * h / bounds.Dx()
	} else {
		w = bounds.Dx() * w / bounds.Dy()
	}
	img = resize.Resize(img, img.Bounds(), w, h)

	m, err := os.Create(path.Join(DATA_DIR, UPLOADS, fn))
	if err != nil {
		Log("Could not create image file")
		return nil, err
	}
	defer m.Close()

	// write new image to file
	err = jpeg.Encode(m, img, nil)
	if err != nil {
		Log("Could not encode image file")
		return nil, err
	}

	return img, err
}

/*func (p *Photo) UpdatezPhoto(db *mgo.Database) error {
	if db == nil {
		db = db_session.Clone().DB(database)
		defer db.Session.Close()
	}
	_, err := db.C("photos").UpsertId(p.Id, p)
	return err
}*/

func (p *Photo) CommentList() []*Comment {
	return p.Comments
}

var (
	UploadForm = forms.Form{
		Fields: []forms.Field{
			forms.Field{Name: "title", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "description"},
			forms.Field{Name: "country", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "location", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "gender", Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "age", Converter: forms.IntConverter, Validators: []forms.Validator{forms.NonemptyValidator}},
			forms.Field{Name: "active", Converter: forms.BoolConverter},
			forms.Field{Name: "photo"},
		},
	}
)
