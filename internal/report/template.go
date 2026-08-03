package report

const tmpl = `
<!DOCTYPE html>
<html>
    <head>
        <meta charset="utf-8">
        <title>STTQ-отчет</title>
        <style>
            body {font-family: monospace; white-space: pre; background-color: black; padding: auto;}
            h1, h2 {color: #2c2b2b;}
            table { width: 100%; border-collapse: collapse; margin: 10px 0;}
            th { background: #e0e0e0; text-align: left; padding: 8px; }
            td { padding: 8px; border-bottom: 1px solid #ddd; }
            .card { background: white; padding: 15px; border-radius: 5px; margin-bottom: 15px; border: 1px solid #ddd; }
        </style>
    </head>
    <body>
        <h1>STTQ Отчет</h1>

        <div class="card">
            <h2>Общая сводка</h2>
            <p><b>Записей:</b>{{.Summary.TotalRecords}}</p>
            <p><b>Успешных:</b>{{.Summary.SuccessfulResults}}</p>
            <p><b>Coverage:</b>{{printf "%1f%%"(mul .Summary.Coverage 100)}}</p>
            <p><b>WER</b>{{.Summary.TotalRecords}}</p>
            <p><b>CER</b>{{.Summary.TotalRecords}}</p>
            <p><b>RTF</b>{{.Summary.TotalRecords}}</p>
            <p><b>Точных совпадений: </b>{{.Summary.ExactMatches}}</p>
        </div>
        
    </body>
</html>`
