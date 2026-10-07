# Background

This document is used as aria/accessibility(a11y) reference for future developers.

There are a lot of a11y problems in the Fomantic UI library. Files in 
`web_src/js/modules/fomantic/` are used as a workaround to make the UI more accessible.

The aria-related code is designed to avoid touching the official Fomantic UI library,
and to be as independent as possible, so it can be easily modified/removed in the future.

To test the aria/accessibility with screen readers, developers can use the following steps:

* On macOS, you can use VoiceOver.
  * Press `Command + F5` to turn on VoiceOver.
  * Try to operate the UI with keyboard-only.
  * Use Tab/Shift+Tab to switch focus between elements.
  * Arrow keys (Option+Up/Down) to navigate between menu/combobox items (only aria-active, not really focused).
  * Press Enter to trigger the aria-active element.
* On Android, you can use TalkBack.
  * Go to Settings -> Accessibility -> TalkBack, turn it on.
  * Long-press or press+swipe to switch the aria-active element (not really focused).
  * Double-tap means old single-tap on the aria-active element.
  * Double-finger swipe means old single-finger swipe.
* TODO: on Windows, on Linux, on iOS

# Known Problems

* Tested with Apple VoiceOver: If a dropdown menu/combobox is opened by mouse click, then arrow keys don't work.
  But if the dropdown is opened by keyboard Tab, then arrow keys work, and from then on, the keys almost work with mouse click too.
  The clue: when the dropdown is only opened by mouse click, VoiceOver doesn't send 'keydown' events of arrow keys to the DOM,
  VoiceOver expects to use arrow keys to navigate between some elements, but it couldn't.
  Users could use Option+ArrowKeys to navigate between menu/combobox items or selection labels if the menu/combobox is opened by mouse click.

# Checkbox

## Accessibility-friendly Checkbox

The ideal checkboxes should be:

```html
<label><input type="checkbox"> ... </label>
```

However, the templates still have the Fomantic-style HTML layout:

```html
<div class="ui checkbox">
  <input type="checkbox">
  <label>...</label>
</div>
```

We call `initAriaLabels` to link the `input` and `label` which makes clicking the
label etc. work. There is still a problem: These checkboxes are not friendly to screen readers,
so we add IDs to all the Fomantic UI checkboxes automatically by JS. If the `label` part is empty,
then the checkbox needs to get the `aria-label` attribute manually.

# The `<aria-menu>` Web Component

New action menus (a button that opens a list of choices, with no search input and no selection state)
should use `<aria-menu>` from `web_src/js/webcomponents/aria-menu.ts` instead of a Fomantic dropdown.
It follows the ARIA menu button pattern and moves real focus onto the items (roving `tabindex`),
so the VoiceOver arrow key problem above cannot happen. Try it on the `/devtest/aria-menu` page.

```html
<aria-menu>
  <button type="button">Actions</button> <!-- first non-.menu child is the trigger -->
  <div class="menu">
    <a class="item" href="...">...</a>
    <div class="divider"></div>
    <div class="item">...</div> <!-- Enter, Space and click all fire the item's "click" -->
  </div>
</aria-menu>
```

Comboboxes, searchable and multiple-selection dropdowns still use Fomantic Dropdown below.

# Fomantic Dropdown

Fomantic Dropdown is designed to be used for many purposes:

* Menu (the profile menu in navbar, the language menu in footer)
* Popup (the branch/tag panel, the review box)
* Simple `<select>` , used in many forms
* Searchable option-list with static items (used in many forms)
* Searchable option-list with dynamic items (ajax)
* Searchable multiple selection option-list with dynamic items: the repo topic setting
* More complex usages, like the Issue Label selector

Fomantic Dropdown requires that the focus must be on its primary element.
If the focus changes, it hides or panics.

At the moment, the aria-related code only tries to partially resolve the a11y problems for dropdowns with items.

There are different solutions:

* combobox + listbox + option:
  * https://www.w3.org/WAI/ARIA/apg/patterns/combobox/
  * A combobox is an input widget with an associated popup that enables users to select a value for the combobox from
    a collection of possible values. In some implementations, the popup presents allowed values, while in other implementations,
    the popup presents suggested values, and users may either select one of the suggestions or type a value.
* menu + menuitem:
  * https://www.w3.org/WAI/ARIA/apg/patterns/menubar/
  * A menu is a widget that offers a list of choices to the user, such as a set of actions or functions.

The current approach is: detect if the dropdown has an input,
if yes, it works like a combobox, otherwise it works like a menu.
Multiple selection dropdowns (`ui multiple ... dropdown`) are partially supported:

* the listbox has `aria-multiselectable="true"` and every option has `aria-selected`,
  refreshed after each add or remove (Fomantic marks the chosen items `active`)
* adding or removing an item is announced through the shared live region in
  `web_src/js/modules/aria-announce.ts`; Fomantic doesn't call `onAdd`/`onRemove` on the
  initial load, so existing selections are not read out
* a selection label's delete icon is named after the label's visible text
* picking an item keeps the keyboard focus: a `GITEA-PATCH` in `web_src/fomantic/build/components/dropdown.js`
  stops Fomantic's IE11 workaround from blurring the dropdown itself, and the global Enter
  quick-submit (`web_src/js/features/common-form.ts`) skips an Enter the dropdown already handled

Still not working: moving between selection labels with Left/Right is not announced, because
`aria-activedescendant` only ever points at menu items.

The issue sidebar's label, assignee, reviewer and project pickers are not Fomantic multiple
dropdowns: selection there is the `checked` class managed by
`web_src/js/features/repo-issue-sidebar-combolist.ts`, which keeps `aria-selected` and the
announcements in step itself. Their items sit in a nested `.scrolling.menu`, which the patch
also covers.

The Fomantic part of this is temporary by design: it goes away when these dropdowns move off
Fomantic (`<aria-menu>` above is for action menus only, so it is not their replacement), while the
sidebar part stays unless the sidebar itself is rewritten.

Some important pages for dropdown testing:

* Home(dashboard) page, the "Create Repo" / "Profile" / "Language" menu.
* Create New Repo page, a lot of dropdowns as combobox.
* Collaborators page, the "permission" dropdown (the old behavior was not quite good, it just works).

```html
<!-- read-only dropdown -->
<div class="ui dropdown"> <!-- focused here, then it's not perfect to use aria-activedescendant to point to the menu item -->
  <input type="hidden" ...>
  <div class="text">Default</div>
  <div class="menu" tabindex="-1"> <!-- "transition hidden|visible" classes will be added by $.dropdown() and when the dropdown is working -->
    <div class="item active selected">Default</div>
    <div class="item">...</div>
  </div>
</div>

<!-- search input dropdown -->
<div class="ui dropdown">
  <input type="hidden" ...>
  <input class="search" autocomplete="off" tabindex="0"> <!-- focused here -->
  <div class="text"></div>
  <div class="menu" tabindex="-1"> <!-- "transition hidden|visible" classes will be added by $.dropdown() and when the dropdown is working -->
    <div class="item selected">...</div>
    <div class="item">...</div>
  </div>
</div>
```
