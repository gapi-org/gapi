package docsui

import (
	"io"
	"net/http"
)

func Serve(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Gapi Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({ url: "/openapi.json", dom_id: "#swagger-ui" });
  </script>
</body>
</html>`)
}

func ServeReDoc(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Gapi ReDoc</title>
</head>
<body>
  <redoc spec-url="/openapi.json"></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`)
}

func ServeScalar(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Gapi Scalar</title>
</head>
<body>
  <script
    id="api-reference"
    data-url="/openapi.json"
    src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`)
}
