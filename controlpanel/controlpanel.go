// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//
// Adapted from tick-tock.go by Nirbhay Choubey
// http://nirbhay.in/2013/03/ajax-with-go/
package controlpanel

// RULES!
//
// The Control Panel is a PUSH model.  DO NOT make any reference
// in this package to anything in Factom, Factoids, or any other
// repository used by Factom!
//
// This allows ANY package to include the controlpanel, and post
// information to be displayed.
//
// CP is the control panel.  If you need to put different information,
// or different information handling, then expand the interface and
// the underlying struct.
//

import (
    "time"
    "strings"
    "fmt"
)

var _ = fmt.Print
var CP = new(ControlPanel)

type IControlPanel interface {
    
    SetPort(string)
    GetPort() string
    SetTitle(string)
    GetTitle() string
        
    AddUpdate  (tag string, cat string, title string, message string, seconds int) // show for int seconds
    Updates()  ([]CPEntry)
    
    Purge()             // Purge system, status, info, warnings, and errors
    
    LastCommunication() time.Time
    
}

// We display what is going on now.  cpEntry's are perged now and again.
//
type CPEntry struct {
    tag    string   // Tagged messages are updated rather than added to
                    //   a list.  This prevents the same message from flooding
                    //   the information presented to the user.
    cat    string   // Catagory (system, status, info, warning, errors)                
    title  string   // Title for this information 
    msg    string   // Message to display
    time   int64    // Time we got it.  We perge every now and then. 
                    // See logs for persistent errors and warnings.
    remove int64    // Time to remove
}

type ControlPanel struct {
    IControlPanel

    running  bool       // Set to true if the control panel is running.
    
    title    string     // Goes in the Tab in the Browser
    port     string     // The port where the control panel is published.
    
    updates  []CPEntry  // Report is processed by Category (status, info, warnings, errors)        
    
    lastCommunication time.Time     // The last time the application updated data on the app.
}

func (cp *ControlPanel) SetPort(port string) {
    cp.port = port
}

func (cp *ControlPanel) GetPort() string {
    if len(cp.port)==0 {
        cp.port = "8090"    // Default to Factom Control Panel
    }
    return cp.port 
}


func (cp *ControlPanel) SetTitle(title string) {
    cp.title = title
}
func (cp *ControlPanel) GetTitle() string {
    if len(cp.title)==0 {
        cp.title = "Factom Control Panel    "    // Default to Factom Control Panel
    }
    return cp.title 
}

func (cp *ControlPanel) Run() {
    cp.lastCommunication = time.Now()
    if cp.running {return}
    cp.running = true
    go runPanel()
}

// Purge our lists of out of date info, warning, and error posts.
func (cp *ControlPanel) Purge() {
    now := time.Now().Unix()
    tags := make(map[string]bool,0)
   
    u := make([]CPEntry,0)
    for i:=len(cp.updates)-1; i>=0; i-- {
        item := cp.updates[i]
        if now < item.remove && !tags[item.tag] {
            u = append(u,item)
        }
        tags[item.tag]=true
    }
    cp.updates = u
}

func (cp *ControlPanel) LastCommunication() time.Time {
    return cp.lastCommunication
}

func (cp *ControlPanel) AddUpdate(tag string, cat string, title string, msg string, expire int) {
    cp.Run()
    cat   = strings.TrimSpace(cat)
    title = strings.TrimSpace(title)
    msg   = strings.TrimSpace(msg)
    msg   = strings.Replace(msg,"\n","<br>",-1)
    if expire <= 0 {expire=9999999}
    cp.updates = append (cp.updates, CPEntry{ 
        tag:    tag,
        cat:    cat,
        title:  title,
        msg:    msg, 
        time:   time.Now().Unix(), 
        remove: time.Now().Unix() + int64(expire),
    })
}

func (cp *ControlPanel) Updates() ([]CPEntry) {
    return cp.updates
}


