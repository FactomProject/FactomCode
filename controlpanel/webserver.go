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
    "log"
    "net/http"
    "time"
)

var _ = time.Sleep

// Content for the control panel html page..
var page =
        `<html>
           <head>
             <link href="data:image/x-icon;base64,AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEJjUbGbP5/wAAAAAAAAAAxYwU/wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACaZjX/oW48/wAAAAAAAAAAHKv5/xyo+P8AAAAA0qRb/wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAI9ZJP8AAAAAAAAAAAAAAACndkP/sIBN/7eKWP8AAAAAHqL3/x6f9/8AAAAA4MOa/wAAAAAAAAAAAAAAALByDP+0dw3/uX0U/7+EFf/FjBT/ypMg/wAAAAAAAAAAvpVl/8Wgcv8AAAAAHpr3/25hTDAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMqTIP/PnEb/1Kdj/wAAAADFoHL/zKt//wAAAAAflfb/AAAAAAAAAAAUuvrxF7b6/xmz+f8asPn/G635/xyq+f8cp/jWAAAAANSnY//ZsXr/AAAAAMyrf//SuI7/AAAAACCP9v8AAAAAF7f6/wAAAAAAAAAAAAAAAAAAAAAPgsM5HaT4/x6i9/8AAAAAAAAAAN29kP8AAAAA0riO/9rFnP8gjfb/AAAAAI5YI/+VYC7/nGk5/6VzQP+vf03/uIxa/wAAAAAdn/f/Hpz3/x6a9v8AAAAA48qm/2xlVzzaxZz/AAAAACCI9f+OWCP/AAAAAAAAAAAAAAAAAAAAALiMWv/Bmmv/yad61QAAAAAfl/b/BB48BOPKpv/q2b3/AAAAAAAAAAAghfX/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMqoe//St47/AAAAACCR9f8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA0reO/wAAAAAgj/b/IIz1/wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAANK3jv/cyaD/AAAAACCJ9f8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA//8AAP2/AADzLwAA3EsAAIGXAAD8SwAAASUAAHzRAAACKgAAeKYAAP5fAAD/TwAA/y8AAP//AAD//wAA//8AAA==" rel="icon" type="image/x-icon" />
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
             <h2>Factom Control Panel</h2>
             <div id="output"></div>
             <script type="text/javascript">
               $(document).ready(function () {
                 $("#output").append("Waiting on Factom...");
                 setInterval("delayedPost()", 1000);
               });
               
               document.title = "factomd"
               
               function delayedPost() {
                 $.post("http://localhost:8090/getreport", "", function(data, status) {
                 $("#output").empty();
                 $("#output").append(data);
                 });
               }
             </script>
           </body>
         </html>`

// handler for the main page.
func handler(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, page)
}

// Build the report to show on the web page
// Standard html (like <br> etc.) can be used.
func handlerGetReport(w http.ResponseWriter, r *http.Request) {
    var out bytes.Buffer
    if len(CP.FactomMode())>0 {
        out.WriteString("Running as "+CP.FactomMode()+"<br>")
    }else{
        out.WriteString("Running Factom<br>")
    }
    if CP.PeriodMark() > 0 && CP.PeriodMark() <= 10 {
        out.WriteString(fmt.Sprintf("Minute %d, Block Height %d<br>",CP.PeriodMark(),CP.BlockHeight()))
    }
    
    out.WriteString(fmt.Sprintf("Number of Transactions Processed: %d<br>",CP.TransactionsProcessed()))
    
    if len(CP.Warnings()) > 0 {
        out.WriteString("<br><b>Warnings</b><br><OL>")
        for _,warn := range CP.Warnings() {
            out.WriteString("<LI>")
            out.WriteString(warn.msg)
        }
        out.WriteString("</OL>")
    }

    if len(CP.Errors()) > 0 {
        out.WriteString("<br><b>Warnings</b><br><OL>")
        for _,err := range CP.Errors() {
            out.WriteString("<LI>")
            out.WriteString(err.msg)
        }
        out.WriteString("</OL>")
    }
    
    
    fmt.Fprint(w, string(out.Bytes()))
    //fmt.Fprint(w, "Running as Server<br>Minute 1, Block Height 123")
}

func runPanel() {
        http.HandleFunc("/controlpanel", handler)
        http.HandleFunc("/getreport", handlerGetReport)
        log.Fatal(http.ListenAndServe(":8090", nil))
}