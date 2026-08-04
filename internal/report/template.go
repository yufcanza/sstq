package report

const tmpl = `<!DOCTYPE html>
<html>
    <head>
        <meta charset="utf-8">
        <title>STTQ-отчет</title>
        <style>
            body {font-family: monospace; background-color: #fafafa; padding: auto;}
            h1, h2 {color: #2c2b2b;}
            table { width: 100%; border-collapse: collapse; margin: 10px 0;}
            th { background: #e0e0e0; text-align: left; padding: auto; }
            td { padding: auto; border-bottom: 1px solid #ddd; }
            .worst { background: #fff3cd; }
            .record { border: 1px solid #ddd; padding: auto; margin: 15px 0; border-radius: 5px; background: white; }
            .card { background: white; padding: auto; border-radius: 5px; margin-bottom: 15px; border: 1px solid #ddd; }
            .match { color: green; }
            .mismatch { color: red; }
            .texts { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin: 10px 0; }
            .metrics { display: flex; gap: 15px; flex-wrap: wrap; font-size: 14px; }
            .badge { display: inline-block; padding: auto; border-radius: 12px; font-size: 12px; margin: 2px; }
            .equal { background: #d4edda; }
            .substitute { background: #fff3cd; }
            .delete { background: #f8d7da; }
            .insert { background: #d1ecf1; }
        </style>
    </head>
    <body>
        <h1>STTQ Отчет</h1>

        <div class="card">
            <h2>Общая сводка</h2>
            <p><b>Записей: </b>{{.Summary.TotalRecords}}</p>
            <p><b>Успешных: </b>{{.Summary.SuccessfulResults}}</p>
            <p><b>Coverage: </b>{{percent .Summary.Coverage}}</p>
            <p><b>WER: </b>{{.Summary.WER}}</p>
            <p><b>CER: </b>{{.Summary.CER}}</p>
            <p><b>RTF: </b>{{.Summary.RTF}}</p>
            <p><b>Точных совпадений: </b>{{.Summary.ExactMatches}}</p>
        </div>
        <div class="card">
            <h2>По тегам</h2>
            {{range $tag, $s := .Groups.ByTag}}
            <p><b>{{$tag}}:</b> samples = {{$s.Samples}}, WER = {{printf "%4f" $s.WER}}, CER= {{printf "%4f" $s.CER}}</p>
            {{else}}<p>Нет данных</p>{{end}}
        </div>
         <div class="card">
            <h2>По длительности</h2>
            {{range $group, $s := .Groups.ByDuration}}
            <p><b>{{$group}}: </b>samples = {{$s.Samples}}, WER = {{printf "%4f" $s.WER}}, CER= {{printf "%4f" $s.CER}}</p>
            {{else}}<p>Нет данных</p>{{end}}
        </div>
        <div class="card">
            <h2>Ошибки</h2>
            {{range .Errors}}
            <p><b>{{.Code}}: </b>{{.Message}} {{if .ID}}(ID: {{.ID}}){{end}}</p>
        </div>
        {{end}}
        <div class="card">
            <h2>Худшие результаты</h2>
            <table>
                <tr><th>ID</th><th>WER</th><th>CER</th><th>Слова</th></tr>
                {{range $i, $r := .Records}}
                {{if lt $i 10}}
                <tr class="worst">
                    <td>{{$r.ID}}</td>
                    <td>{{printf "%.4f" $r.WER}}</td>
                    <td>{{printf "%.4f" $r.CER}}</td>
                    <td>{{$r.ReferenceWords}}</td>
                </tr>
                {{end}}
                {{end}}
            </table>
        </div>
        
        <div class="card">
            <h2>Результаты</h2>
            {{range.Records}}
            <div class="record">
                <p><b>{{.ID}}:</b><span class="{{if .ExactMatch}}match{{else}}mismatch{{end}}">
                    {{if .ExactMatch}}Совпадение{{else}}Ошибка{{end}}
                </span></p>
                <div class="texts">
                    <div><b>Эталон:</b>{{.Reference}}</div>
                    <div><b>Гипотеза:</b>{{.Hypothesis}}</div>
                </div>
                <div class="metrics">
                    <span>WER: {{printf "%.4f" .WER}}</span>
                    <span>CER: {{printf "%.4f" .CER}}</span>
                    <span>Совпало: {{.Hits}}</span>
                    <span>Замен: {{.Substitutions}}</span>
                    <span>Удалений: {{.Deletions}}</span>
                    <span>Вставок: {{.Insertions}}</span>
                    {{if .Tags}}<span>Теги: {{range .Tags}}{{.}}{{end}}</span>{{end}}
                </div>
                {{if .Alignment}}
                <div class="alignment">
                    {{range .Alignment}}
                    <span class="badge {{.Type}}">
                        {{.Manifest}}->{{.Hypothesis}}
                    </span>
                    {{end}}
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
    </body>
</html>`
