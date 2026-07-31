package main

import (
	"html/template"
	"os"
	"time"
)

type reportData struct {
	Movies    []movie
	UpdatedAt string
}

var reportTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"year": func(value *int) any {
		if value == nil {
			return "—"
		}
		return *value
	},
	"rating": func(value *float64) any {
		if value == nil {
			return "—"
		}
		return *value
	},
}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Riga Cinemas — Now Playing</title>
<style>
:root{
  color-scheme:light;
  --ink:#13231b;
  --muted:#68766e;
  --green:#128454;
  --green-dark:#09633d;
  --green-soft:#e5f7ed;
  --line:#e7ece9;
  --surface:#fff;
}
*{box-sizing:border-box}
body{
  margin:0;
  min-height:100vh;
  color:var(--ink);
  font:15px/1.5 Inter,ui-sans-serif,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
  background:
    radial-gradient(circle at 82% 0%,rgba(122,214,161,.24),transparent 34rem),
    linear-gradient(180deg,#eef9f2 0,#f7f8f6 31rem);
}
body::before{
  content:"";
  position:fixed;
  inset:0;
  pointer-events:none;
  opacity:.3;
  background-image:radial-gradient(#9fb7a8 .55px,transparent .55px);
  background-size:12px 12px;
  mask-image:linear-gradient(to bottom,black,transparent 42rem);
}
main{position:relative;width:min(1180px,calc(100% - 40px));margin:0 auto;padding:56px 0 72px}
.eyebrow{
  display:flex;
  align-items:center;
  gap:9px;
  margin-bottom:20px;
  color:var(--green-dark);
  font-size:12px;
  font-weight:800;
  letter-spacing:.12em;
  text-transform:uppercase;
}
.eyebrow::before{content:"";width:8px;height:8px;border-radius:50%;background:#2fbd79;box-shadow:0 0 0 5px rgba(47,189,121,.12)}
.hero{display:flex;align-items:end;justify-content:space-between;gap:32px;margin-bottom:34px}
h1{max-width:760px;margin:0;font-size:clamp(40px,6.5vw,76px);line-height:.97;letter-spacing:-.065em;font-weight:760}
.accent{color:var(--green)}
.lede{max-width:580px;margin:19px 0 0;color:var(--muted);font-size:17px}
.count-card{
  flex:0 0 auto;
  min-width:160px;
  padding:20px 22px;
  border:1px solid rgba(18,132,84,.14);
  border-radius:22px;
  background:rgba(255,255,255,.68);
  box-shadow:0 18px 55px rgba(30,80,52,.08);
  backdrop-filter:blur(12px);
}
.count-card strong{display:block;font-size:34px;line-height:1;font-weight:780;letter-spacing:-.04em}
.count-card span{display:block;margin-top:8px;color:var(--muted);font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:.09em}
.panel{
  overflow:hidden;
  border:1px solid rgba(30,61,44,.1);
  border-radius:24px;
  background:var(--surface);
  box-shadow:0 26px 80px rgba(31,67,46,.11),0 2px 8px rgba(31,67,46,.04);
}
.toolbar{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:20px 22px;border-bottom:1px solid var(--line)}
.toolbar-title{font-size:13px;font-weight:750}
.updated{margin-left:8px;color:var(--muted);font-weight:500}
.search{position:relative;width:min(100%,300px)}
.search::before{content:"⌕";position:absolute;left:14px;top:50%;translate:0 -53%;color:#75847b;font-size:22px;pointer-events:none}
.search input{
  width:100%;
  height:42px;
  padding:0 14px 0 41px;
  border:1px solid #dfe7e2;
  border-radius:13px;
  outline:none;
  background:#f8faf9;
  color:var(--ink);
  font:inherit;
  transition:.2s ease;
}
.search input:focus{border-color:#76caa1;background:#fff;box-shadow:0 0 0 4px rgba(18,132,84,.09)}
.table-wrap{overflow:auto}
table{width:100%;border-collapse:collapse}
th,td{padding:17px 20px;text-align:left;border-bottom:1px solid var(--line);white-space:nowrap}
th{
  position:sticky;
  top:0;
  z-index:1;
  background:#fbfcfb;
  color:#77837c;
  font-size:11px;
  font-weight:800;
  text-transform:uppercase;
  letter-spacing:.1em;
}
th.sortable{cursor:pointer;user-select:none}
th.sortable::after{content:"↕";margin-left:7px;color:#b5c0b9;font-size:12px}
th[data-direction="asc"]::after{content:"↑";color:var(--green)}
th[data-direction="desc"]::after{content:"↓";color:var(--green)}
tbody tr{transition:background .18s ease}
tbody tr:hover{background:#f8fcfa}
tbody tr:last-child td{border-bottom:0}
.movie-title{font-weight:720;letter-spacing:-.012em}
.year,.genres{color:var(--muted)}
.cinema-link{
  display:inline-flex;
  align-items:center;
  justify-content:center;
  min-width:66px;
  padding:6px 10px;
  border-radius:9px;
  font-size:12px;
  font-weight:800;
  text-decoration:none;
  transition:transform .16s ease,box-shadow .16s ease;
}
.cinema-link::after{content:"↗";margin-left:5px;font-size:10px}
.cinema-link:hover{transform:translateY(-1px)}
.forum-link{background:#e5f7ed;color:#09633d}
.forum-link:hover{box-shadow:0 5px 14px rgba(18,132,84,.16)}
.apollo-link{background:#f1eafe;color:#5b2ca0}
.apollo-link:hover{box-shadow:0 5px 14px rgba(91,44,160,.16)}
.score{
  display:inline-flex;
  align-items:center;
  min-width:43px;
  justify-content:center;
  padding:5px 9px;
  border-radius:9px;
  background:var(--green-soft);
  color:var(--green-dark);
  font-size:13px;
  font-weight:820;
  font-variant-numeric:tabular-nums;
}
.empty{color:#a8b2ac}
.imdb{
  display:inline-flex;
  align-items:center;
  gap:4px;
  color:#30463a;
  font-size:13px;
  font-weight:750;
  text-decoration:none;
}
.imdb::after{content:"↗";color:#91a098;font-size:11px;transition:transform .16s ease}
.imdb:hover{color:var(--green)}
.imdb:hover::after{transform:translate(2px,-2px)}
.no-results{display:none;padding:42px;text-align:center;color:var(--muted)}
@media(max-width:760px){
  main{width:min(100% - 24px,1180px);padding:32px 0 48px}
  .hero{align-items:start;flex-direction:column}
  .count-card{min-width:0;width:100%}
  .toolbar{align-items:stretch;flex-direction:column}
  .search{width:100%;max-width:none}
  th,td{padding:15px 16px}
}
</style>
</head>
<body>
<main>
<div class="eyebrow">Riga cinemas · now playing</div>
<header class="hero">
  <div>
    <h1>Pick your next <span class="accent">great movie.</span></h1>
    <p class="lede">Forum Cinemas listings, IMDb scores, and direct links to Forum and Apollo Akropole.</p>
  </div>
  <div class="count-card"><strong>{{len .Movies}}</strong><span>movies playing</span></div>
</header>
<section class="panel">
<div class="toolbar">
  <div class="toolbar-title">Now playing <span class="updated">Updated {{.UpdatedAt}}</span></div>
  <label class="search"><input id="search" type="search" placeholder="Search movies or genres…" autocomplete="off"></label>
</div>
<div class="table-wrap">
<table id="movies">
<thead><tr><th class="sortable">Title</th><th>Forum</th><th>Apollo</th><th class="sortable">Year</th><th class="sortable">Rating</th><th>IMDb</th><th class="sortable">Genres</th></tr></thead>
<tbody>
{{range .Movies}}<tr>
  <td class="movie-title">{{.Title}}</td>
  <td><a class="cinema-link forum-link" href="{{.ForumCinemasURL}}" target="_blank" rel="noopener">Forum</a></td>
  <td>{{if .ApolloKinoURL}}<a class="cinema-link apollo-link" href="{{.ApolloKinoURL}}" target="_blank" rel="noopener">Apollo</a>{{else}}<span class="empty">—</span>{{end}}</td>
  <td class="year">{{year .ReleaseYear}}</td>
  <td>{{if .IMDbRating}}<span class="score">{{rating .IMDbRating}}</span>{{else}}<span class="empty">—</span>{{end}}</td>
  <td><a class="imdb" href="{{.IMDbURL}}" target="_blank" rel="noopener">IMDb</a></td>
  <td class="genres">{{.Genres}}</td>
</tr>{{end}}
</tbody>
</table>
</div>
<div class="no-results" id="no-results">No movies found. Try another search.</div>
</section>
</main>
<script>
const table=document.querySelector("#movies"),body=table.querySelector("tbody"),rows=[...body.rows];
document.querySelectorAll("th.sortable").forEach(th=>th.addEventListener("click",()=>{
  const n=th.cellIndex;
  const asc=th.dataset.direction!=="asc";
  document.querySelectorAll("th.sortable").forEach(header=>delete header.dataset.direction);
  rows.sort((a,b)=>a.cells[n].innerText.localeCompare(b.cells[n].innerText,undefined,{numeric:true})*(asc?1:-1));
  rows.forEach(row=>body.append(row));
  th.dataset.direction=asc?"asc":"desc";
}));
document.querySelector("#search").addEventListener("input",event=>{
  const query=event.target.value.trim().toLocaleLowerCase();
  let visible=0;
  rows.forEach(row=>{
    const match=!query||row.innerText.toLocaleLowerCase().includes(query);
    row.hidden=!match;
    visible+=match;
  });
  document.querySelector("#no-results").style.display=visible?"none":"block";
});
</script>
</body>
</html>`))

func writeReport(path string, movies []movie, now time.Time) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return reportTemplate.Execute(file, reportData{
		Movies:    movies,
		UpdatedAt: now.Format("02 Jan 2006"),
	})
}
