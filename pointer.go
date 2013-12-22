package pointer

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"strings"

	"appengine"
	"appengine/channel"
	"appengine/memcache"
)

const key = "ids"

func init() {
	http.HandleFunc("/", main)
	http.HandleFunc("/update", update)
	http.HandleFunc("/leave", leave)
}

func main(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	ids := getIDs(c)
	id := fmt.Sprintf("%d", rand.Int())
	found := false
	for _, mid := range ids {
		if id == mid {
			found = true
		}
	}
	if !found {
		ids = append(ids, id)
		_ = memcache.Set(c, &memcache.Item{
			Key:   key,
			Value: []byte(strings.Join(ids, ",")),
		})
	}

	tok, _ := channel.Create(c, id)
	tmpl.Execute(w, struct{ Token, ClientID string }{tok, id})
}

func update(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	ids := getIDs(c)
	x := r.FormValue("x")
	y := r.FormValue("y")
	from := r.FormValue("id")
	msg := fmt.Sprintf("{\"x\":%s,\"y\":%s,\"id\":\"%s\"}", x, y, from)
	for _, id := range ids {
		if id != from {
			_ = channel.Send(c, id, msg)
		}
	}
}

func leave(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	ids := getIDs(c)
	i := -1
	id := r.FormValue("id")
	for ni, mid := range ids {
		if id == mid {
			i = ni
			break
		}
	}
	if i != -1 {
		ids = append(ids[:i], ids[i+1:]...)
		_ = memcache.Set(c, &memcache.Item{
			Key:   key,
			Value: []byte(strings.Join(ids, ",")),
		})
	}
	msg := fmt.Sprintf("{\"x\":-1,\"y\":-1,\"id\":\"%s\"}", id)
	for _, id := range ids {
		_ = channel.Send(c, id, msg)
	}
}

func getIDs(c appengine.Context) []string {
	var ids []string
	if i, err := memcache.Get(c, key); err == nil {
		ids = strings.Split(string(i.Value), ",")
	}
	return ids
}

var tmpl = template.Must(template.New("n").Parse(`
<!doctype html>
<html style="height:100%;">
  <head>
    <title>Pointer Party!</title>
    <script src='/_ah/channel/jsapi'></script>
  </head>
  <body style="height:100%;font-family:Arial;font-size:10px;">
    <script type="text/javascript">
      var my_id = '{{.ClientID }}';
      var checkEvery = 200; //ms
      var body = document.getElementsByTagName('body')[0];

      var channel = new goog.appengine.Channel('{{.Token}}');
      var socket = channel.open();

      socket.onmessage = function(response) {
        var data = JSON.parse(response.data);
        var x = data.x;
        var y = data.y;
        var id = data.id;
        // Update the correct pointer's location, or show a new pointer
        // console.log('receiving: x=' + x + ' y=' + y + ' id=' + id)
        var img = document.getElementById(id);
        if (img == null) {
          img = new Image();
          img.id = id;
          img.src = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAwAAAAVCAYAAAByrA+0AAAAAXNSR0IArs4c6QAAAAZiS0dEAP8A/wD/oL2nkwAAAAlwSFlzAAALEwAACxMBAJqcGAAAAAd0SU1FB9gFAhEpAuf8RJkAAAAddEVYdENvbW1lbnQAQ3JlYXRlZCB3aXRoIFRoZSBHSU1Q72QlbgAAAHNJREFUOMu1k9sKwCAMQ5uw///l7GEOOrUXH1YQteRYG9HMTHYQHLOOAElt6K3Qhug3HYhzooK4S2YQo5MiiNl9dxArV2aIHe89dGVCAAvLhgh+sBDHtg4xniVyl5y4XQFrKwhtxckHQv0E+vRI+yHke7gBphY1FrmNcUIAAAAASUVORK5CYII=';
          img.style.position = 'absolute';
          body.appendChild(img);
        }
        if (x < 0 || y < 0) {
          img.style.display = 'none';
          img = null;
        } else {
                  img.style.left = x + 'px';
          img.style.top = y + 'px';
        }
      }

      var actualonmousemove = function(ev) {
        body.onmousemove = null;
        setTimeout('body.onmousemove = actualonmousemove', checkEvery);

        // Post the current mouse position to the server.
        var xhr = new XMLHttpRequest();
        var x = ev.clientX;
        var y = ev.clientY;
        var path = '/update?x=' + x + '&y=' + y + '&id=' + my_id;
        // console.log('sending: x=' + x + ' y=' + y);
        xhr.open('POST', path, true);
        xhr.send();
      }
      body.onmousemove = actualonmousemove;

      window.onbeforeunload = function() {
        var xhr = new XMLHttpRequest();
        xhr.open('POST', '/leave?id=' + my_id);
        xhr.send();
      }
    </script>
    <p><b>Welcome to Pointer Party!</b></p>
    <p><b>Directions:</b> Move your mouse pointer. See the mouse pointers of other users on this page.</p>
    <p>Built using <a href="https://developers.google.com/appengine/docs/go/channel/reference">App Engine Channel API</a> (<a href="https://github.com/ImJasonH/pointerparty">src</a>)
  </body>
</html>
`))
