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
var CP IControlPanel = new(ControlPanel)

type IControlPanel interface {
    SetFactomMode(string)
    FactomMode() string
    
    UpdatePeriodMark(int)
    PeriodMark() int
    UpdateBlockHeight(int)
    BlockHeight() int
    
    UpdateTransactionsProcessed(int)
    TransactionsProcessed() int 
    
    AddError(string)
    Errors() ([]CPEntry)
    AddWarning(string)
    Warnings() ([]CPEntry)
    
    LastCommunication() time.Time
    
}

// We display what is going on now.  cpEntry's are perged now and again.
//
type CPEntry struct {
    msg  string     // Message to display
    time int64      // Time we got it.  We perge every now and then. 
                    // See logs for persistent errors and warnings.
}
type ControlPanel struct {
    IControlPanel
    
    factommode      string
    period          int
    blockHeight     int
    numTransactions int
    
    errors   []CPEntry
    warnings []CPEntry
    
    running     bool
    
    lastCommunication time.Time
}

func (cp *ControlPanel) Run() {
    cp.lastCommunication = time.Now()
    if cp.running {return}
    cp.running = true
    go runPanel()
}

func (cp *ControlPanel) LastCommunication() time.Time {
    return cp.lastCommunication
}

func (cp *ControlPanel) AddWarning(warning string) {
    warning = strings.TrimSpace(warning)
    warning = strings.Replace(warning,"\n","<br>",-1)
    cp.warnings = append (cp.warnings, CPEntry{ 
        msg: warning, 
        time: time.Now().Unix(), 
    })
}
func (cp *ControlPanel) Warnings() ([]CPEntry) {
    return cp.warnings
}

func (cp *ControlPanel) AddError(err string) {
    err = strings.TrimSpace(err)
    err = strings.Replace(err,"\n","<br>",-1)
    cp.errors = append (cp.errors, CPEntry{ 
        msg: err, 
        time: time.Now().Unix(), 
    })
}
func (cp *ControlPanel) Errors() ([]CPEntry) {
    return cp.errors
}


func (cp *ControlPanel) UpdateTransactionsProcessed(num int) {
    cp.Run()
    cp.numTransactions = num
}
func (cp *ControlPanel) TransactionsProcessed() int {
    return cp.numTransactions
}

func (cp *ControlPanel) SetFactomMode(mode string) {
    cp.Run()
    cp.factommode = mode
}
func (cp *ControlPanel) FactomMode() string {
    return cp.factommode
}


func (cp *ControlPanel) UpdatePeriodMark(period int) {
    cp.Run()
    cp.period = period
}
func (cp *ControlPanel) PeriodMark() int {
    return cp.period
}

func (cp *ControlPanel) UpdateBlockHeight(height int) {
    if height < 0 {return}
    cp.Run()
    cp.blockHeight = height
}
func (cp *ControlPanel) BlockHeight() int {
    return cp.blockHeight
}
