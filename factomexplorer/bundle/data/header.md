# Factom

Block Explorer

  * {{if eq .Title "Home"}}
      <div>Home</div>
    {{else}}
      [Home](/home)
    {{end}}
  * {{if eq .Title "Entries"}}
      <div>Entries</div>
    {{else}}
      [Entries](/entries)
    {{end}}
    <select id="entry_select" data-entry-id="{{unnil .EntryID -1}}" onchange="window.location='/entries/'+this.item(this.selectedIndex).innerHTML">
      <option selected></option>
      {{range activeEntryIDs}}<option>{{.}}</option>{{end}}
      <option>+</option>
    </select>
  * {{if eq .Title "Chains"}}
      <div>Chains</div>
    {{else}}
      [Chains](/chains)
    {{end}}
    <select id="key_select" data-key-id="{{unnil .KeyID -1}}" onchange="window.location='/chains/'+this.item(this.selectedIndex).innerHTML">
      <option selected></option>
      {{range keyIDs}}<option>{{.}}</option>{{end}}
      <option>+</option>
    </select>
  * {{if eq .Title "Keys"}}
      <div>Keys</div>
    {{else}}
      [Keys](/keys)
    {{end}}
    <select id="key_select" data-key-id="{{unnil .KeyID -1}}" onchange="window.location='/keys/'+this.item(this.selectedIndex).innerHTML">
      <option selected></option>
      {{range keyIDs}}<option>{{.}}</option>{{end}}
      <option>+</option>
    </select>
  * {{if eq .Title "Explore"}}
      <div>Explore</div>
    {{else}}
      [Explore](/explore)
    {{end}}
  * {{if eq .Title "Search"}}
      <div>Search</div>
    {{else}}
      [Search](/search)
    {{end}}
  * [Help](http://github.com/FactomProject/FactomDocs)<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><g transform="translate(-826.429 -698.791)"><rect width="5.982" height="5.982" x="826.929" y="702.309" fill="#ccc" stroke="#666"></rect><g><path d="M831.194 698.791h5.234v5.391l-1.571 1.545-1.31-1.31-2.725 2.725-2.689-2.689 2.808-2.808-1.311-1.311z" fill="#777"></path><path d="M835.424 699.795l.022 4.885-1.817-1.817-2.881 2.881-1.228-1.228 2.881-2.881-1.851-1.851z" fill="#eee"></path></g></g></svg>
