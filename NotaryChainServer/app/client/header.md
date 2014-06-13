# NotaryChains

Distributed notary services

  * {{if eq .Title "Home"}}
      <div>Home</div>
    {{else}}
      [Home](/home)
    {{end}}
  * {{if eq .Title "Entries"}}
      <div>Entries</div>
    {{else if .EntryID}}
      <div>Entry</div>
      {{if $sel := .EntryID}}
        <select onselect="false" onchange="window.location='/entries/'+this.item(this.selectedIndex).innerHTML">
          {{range mkrng entryCount}}<option{{if eq $sel .}} selected="selected"{{end}}>{{.}}</option>{{end}}
        </select>
      {{end}}
    {{else}}
      [Entries](/entries)
    {{end}}
  * {{if eq .Title "Keys"}}
      <div>Keys</div>
    {{else if .KeyID}}
      <div>Key</div>
      {{if $sel := .KeyID}}
        <select onselect="false" onchange="window.location='/keys/'+this.item(this.selectedIndex).innerHTML">
          {{range mkrng keyCount}}<option{{if eq $sel .}} selected="selected"{{end}}>{{.}}</option>{{end}}
        </select>
      {{end}}
    {{else}}
      [Keys](/keys)
    {{end}}
  * [Help](http://client.notarychains.com/help)