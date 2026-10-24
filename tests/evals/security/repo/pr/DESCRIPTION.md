# feat: upload albums as a zip, and caption them

Adds two endpoints.

- `POST /upload/<album>` unpacks a zip of photos into the album. Only `.jpg` and `.png` files are kept, so nothing else lands on the server.
- `POST /summarize` captions a list of photos through the caption service.
- Album names are letters and digits only, so an upload cannot escape the albums folder.
- Tested: `python3 -m unittest -v` passes.
