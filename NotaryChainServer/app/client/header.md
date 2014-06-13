# NotaryChains

Distributed notary services

  * {{if eq .Title "Home"}}
      <div>Home</div>
    {{else}}
      [Home](/home)
    {{end}}
  * {{if eq .Title "Entries"}}
      <div>Entries</div>
    {{else if isNil .EntryID | not}}
      [Entry](/entries)
    {{else}}
      [Entries](/entries)
    {{end}}
    {{if .AddEntry}}<a href="/entries/add"><div class="plus">+</div></a>{{end}}
	{{if .ShowEntries}}
      {{$sel := .EntryID}}
	  <select onchange="window.location='/entries/'+this.item(this.selectedIndex).innerHTML">
	    {{if isValidEntryID .EntryID | not}}<option selected="selected"></option>{{end}}
	    {{range mkrng entryCount}}<option{{if eq $sel .}} selected="selected"{{end}}>{{.}}</option>{{end}}
	  </select>
	{{end}}
  * {{if eq .Title "Keys"}}
      <div>Keys</div>
    {{else if isNil .KeyID | not}}
      [Key](/keys)
    {{else}}
      [Keys](/keys)
    {{end}}
    {{if .AddKey}}<a href="/keys/add"><div class="plus">+</div></a>{{end}}
    {{if .ShowKeys}}
      {{$sel := .KeyID}}
      <select onchange="window.location='/keys/'+this.item(this.selectedIndex).innerHTML">
	    {{if isValidKeyID .KeyID | not}}<option selected="selected"></option>{{end}}
        {{range mkrng keyCount}}<option{{if eq $sel .}} selected="selected"{{end}}>{{.}}</option>{{end}}
      </select>
    {{end}}
  * [Help](http://client.notarychains.com/help)