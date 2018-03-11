package draft

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"

	"github.com/oklog/ulid"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/strvals"

	"github.com/Azure/draft/pkg/draft/local"
	"github.com/Azure/draft/pkg/rpc"
	"github.com/Azure/draft/pkg/storage"
)

// AppContext contains state information carried across the various draft stage boundaries.
type AppContext struct {
	obj  *storage.Object
	srv  *Server
	req  *rpc.UpRequest
	buf  *bytes.Buffer
	tag  string
	img  string
	out  io.Writer
	id   string
	vals chartutil.Values
}

// newAppContext prepares state carried across the various draft stage boundaries.
func newAppContext(s *Server, req *rpc.UpRequest, out io.Writer) (*AppContext, error) {
	raw := bytes.NewBuffer(req.AppArchive.Content)
	// write build context to a buffer so we can also write to the sha1 hash.
	b := new(bytes.Buffer)
	h := sha1.New()
	w := io.MultiWriter(b, h)
	if _, err := io.Copy(w, raw); err != nil {
		return nil, err
	}
	// truncate checksum to the first 40 characters (20 bytes) this is the
	// equivalent of `shasum build.tar.gz | awk '{print $1}'`.
	ctxtID := h.Sum(nil)
	imgtag := fmt.Sprintf("%.20x", ctxtID)
	image := fmt.Sprintf("%s/%s:%s", s.cfg.Registry.URL, req.AppName, imgtag)

	// inject certain values into the chart such as the registry location,
	// the application name, and the application version.
	tplstr := "image.repository=%s/%s,image.tag=%s,basedomain=%s,%s=%s,ingress.enabled=%s"
	inject := fmt.Sprintf(tplstr, s.cfg.Registry.URL, req.AppName, imgtag, s.cfg.Basedomain, local.DraftLabelKey, req.AppName, strconv.FormatBool(s.cfg.IngressEnabled))

	vals, err := chartutil.ReadValues([]byte(req.Values.Raw))
	if err != nil {
		return nil, err
	}
	if err := strvals.ParseInto(inject, vals); err != nil {
		return nil, err
	}
	buildID := getulid()
	return &AppContext{
		obj:  &storage.Object{BuildID: buildID, ContextID: ctxtID},
		id:   buildID,
		srv:  s,
		req:  req,
		buf:  b,
		tag:  imgtag,
		img:  image,
		out:  out,
		vals: vals,
	}, nil
}

func getulid() string { return <-ulidc }

// A channel which returns build ulids.
var ulidc = make(chan string)

func init() {
	rnd := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	go func() {
		for {
			ulidc <- ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rnd).String()
		}
	}()
}
