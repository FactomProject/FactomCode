// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//
// Adapted from tick-tock.go by Nirbhay Choubey
// http://nirbhay.in/2013/03/ajax-with-go/
package controlpanel

import (
    "fmt"
    "bytes"
    "strings"
    "log"
    "net/http"
    "time"
)

var _ = time.Sleep

// Content for the control panel html page..
var page =`
<html>
    <head>
        <title>%s</title>
        <link href="data:image/x-icon;base64,
        AAABAAEAICAAAAEAIACoEAAAFgAAACgAAAAgAAAAQAAAAAEAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/
        AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAA
        AP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/
        AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAA
        AP8AAAD/AAAA/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f/ruCP/67gj/+u4I/8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/xyF9f8chfX/HIX1/xyF9f8chfX/
        HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP/67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/HIX1/xyF
        9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP/67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/xyF9f8chfX/HIX1/xyF9f8chfX/
        HIX1/+u4I//ruCP/67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/////////////////////////////////////////////////ruCP/67gj/+u4I//ruCP/67gj/+u4I/8chfX/HIX1/xyF9f8chfX/HIX1/+u4I//ruCP/67gj/wAA
        AP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP//////////////////////+u4I//ruCP/67gj/+u4I/8chfX/HIX1/xyF9f8chfX/HIX1/+u4I//ruCP/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/
        67gj/+u4I//ruCP/67ch/+u4I//ruCP/67YX/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//////////////////ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/+u4I//ruCP/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/HIX1/xyF9f8chfX/HIX1/xyF
        9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP/67gj/+u5Jv/ruCP////////////ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/+u4I/8AAAD/AAAA/wAAAP8AAAD/7Lok/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/
        HIX1/+u4I//ruCP/67gj/+u4I//ruCP////////////ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/wAAAP8AAAD/AAAA/wAAAP8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f/ruCP/67gj/+u4
        I//ruCP////////////ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/wAAAP8AAAD/AAAA/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP////////////ruCP/
        67gj/+u4I/8chfX/HIX1/xyF9f8chfX/AAAA/wAAAP8AAAD/HIX1/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP////////////ruCP/67gj/+u4I/8chfX/HIX1/xyF
        9f8chfX/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/xyF9f8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP////////////ruCP/67gj/+q1C/8chfX/HIX1/xyF9f8AAAD/AAAA/+u4I//ruCP/
        67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4I/8chfX/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I//ruCP//////+u4I//ruCP/67gj/+u4I/8chfX/HIX1/wAAAP8AAAD/67gj/+u4I//ruCP/67gj/+u4I//ruCP/67gj/+u4
        I//ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/xyF9f8chfX/67gj/+u4I////////////+u4I//ruCP/67gj/xyF9f8chfX/AAAA/wAAAP/ruCP/67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/67gj/+u4I//ruCP/67gj/+u4I//ruCP/
        67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/xyF9f/ruCP/67gj/+u4I///////67gj/+u4I//ruCP/HIX1/xyF9f8AAAD/AAAA/+u4I/8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/HIX1/xyF
        9f8chfX/HIX1/xyF9f/ruCP/67gj////////////67gj/wAAAP8AAAD/HIX1/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/+zAN//ruCP/
        67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/67gj/+u4I//ruCP/67gj/+u4I//ruCP/HIX1/xyF9f8chfX/HIX1/+u3Hv/ruCP/AAAA/wAAAP8AAAD/AAAA/wAA
        AP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/67gj/+u4I//ruCP/67gj/+u4I/8chfX/HIX1/xyF9f8chfX/67gj/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/
        AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/xyF9f8chfX/HIX1/xyF9f8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAA
        AP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP/ruCP/67gj/+u4I//ruCP/67gj/xyF9f8chfX/HIX1/xyF9f8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/
        AAAA/wAAAP8AAAD/67gj/+u4I//ruCP/67gj/xyF9f8chfX/HIX1/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/+u4I+3ruCP/67gj/+u4
        I//ruCP/HIX1/xyF9f8chfX/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/+u4I//ruCP/67gj/wAAAP8AAAD/HIX1/xyF9f8AAAD/
        AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/HIX1/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAA
        AP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/
        AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAA/wAAAP8AAAD/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
        AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" 
        rel="icon" type="image/x-icon" />
        <script type="text/javascript"
        src="http://ajax.googleapis.com/ajax/libs/jquery/1.3.2/jquery.min.js">
        </script>
        <style> 
        div {
            font-family: "Times New Roman", Georgia, Serif;
            font-size: 1em;
            width: 40.3em;
            padding: 8px 8px; 
            border: 2px solid #2B1B17;
            border-radius: 10px;
            color: #2B1B17;
            text-shadow: 1px 1px #E5E4E2;
            background: #FFFFFF;
        }
        </style>
    </head>
    <body>
        <h3>Factom</h3>
        <div id="output">
            <script type="text/javascript">
                $(document).ready(function () {
                    $("#output").append("Waiting on Factom...");
                    setInterval("delayedPost()", 1000);
                });
                            
                function delayedPost() {
                    $.post("http://localhost:%s/getreport", "", function(data, status) {
                        $("#output").empty();
                        $("#output").append(data);
                    });
                }
            </script>
        </div>
    </body>
</html>
`

// handler for the main page.
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, fmt.Sprintf(page,CP.GetTitle(),CP.GetPort()))
}

// Build the report to show on the web page
// Standard html (like <br> etc.) can be used.
func handlerGetReport(w http.ResponseWriter, r *http.Request) {
    var out bytes.Buffer
    since := time.Since(CP.LastCommunication())
    out.WriteString("Last update: ")
    if int(since.Hours())>0 {
        hours := int(since.Hours())
        out.WriteString(fmt.Sprintf("more than %d hour(s) ago<br>",hours))
    }else if int(since.Minutes())>0 {
        minutes := int(since.Minutes())
        out.WriteString(fmt.Sprintf("more than %d minute(s) ago<br>",minutes))
    }else{
        seconds := int(since.Seconds())
        out.WriteString(fmt.Sprintf("%d second(s) ago<br>",seconds))
    }
    
    CP.Purge()
    
    for i:=0; i<len(CP.updates)-1; i++ {
        for j:=0; j<len(CP.updates)-i-1; j++ {
            if CP.updates[j].title > CP.updates[j+1].title {
                t := CP.updates[j]
                CP.updates[j] = CP.updates[j+1]
                CP.updates[j+1] = t
            }
        }
    }
    
    if len(CP.Updates()) > 0 {        
        cats := []string  { "system","status", "info", "warnings", "errors"}
        for _,cat := range cats {
            first := true
            for _,update := range CP.Updates() {
                if update.cat == cat {
                    if first { 
                        out.WriteString(fmt.Sprintf("<br><b>%s</b><br><OL><dl>",strings.Title(cat)))
                        first=false
                    }
                    if len(update.title)>0 {out.WriteString("<dt>"+update.title+"</dt>")}
                    if len(update.msg)>0   {out.WriteString("<dd>"+update.msg+"</dd>")}
                }
            }
            if !first {out.WriteString("</OL></dl>")}
        }
        out.WriteString("</OL>")
    }
    fmt.Fprint(w, string(out.Bytes()))    
}

func handlerGetReport2(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w,"fctWallet report")
}

func runPanel() {
        http.HandleFunc("/controlpanel", handler)
        http.HandleFunc("/getreport", handlerGetReport)
        log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",CP.GetPort()), nil))
}