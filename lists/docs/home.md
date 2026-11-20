+++
URL = ""
Title = "Cogent Lists"
+++

<img src="icon.svg" alt="Cogent Lists icon" width="128">

**Cogent Lists** is an app for all of your lists: to-do lists, shopping lists, projects, reading lists, and anything else. The lists are in groups, in a tree on the left, and the items of the current list are in a table, where they can be in categories, such as the aisles of a store in a shopping list. Each list has a *type*, which gives the fields of its items, the columns of the table, such as Done, Due, and Priority for a to-do list; you can change the types and make new ones. To-do items with due dates can remind you with the notifications of your device, and the lists can be synced with a [[#Syncing with GitHub|GitHub]] repository, so that they are the same on all of your devices, with the history of all of the changes.

Cogent Lists runs on macOS, Windows, Linux, iOS, Android, and the web. On phones, the tree of lists, the items of a list, and the form of an item are each in a full window, in turn, with back buttons.

## Lists and groups

* Use **New list** and **New group** in the toolbar to make a list or a group of lists, in the current group or after the current list. A new list has the type of the current list.
* The context menu of a list (right click, or long press) has **Rename**, **Duplicate**, **Delete**, **Copy**, **Cut**, and **Paste**. Drag lists to move them, into groups or between them.
* Click a list to show its items.

## Items

* Click a cell to edit it. The first column is the tree of the items: an item can have items in it, as a category does.
* The **+** button adds an item after the selected one, and **+ sub** adds an item in it.
* The indent and outdent buttons (`Command+]` and `Command+[`) move the selected items into the item before them, or out of the item they are in.
* Drag items by their first column to move them. Shift and Command (Control) clicks select several items, as does **Select items**, for selecting by tapping on phones.
* **Undo** and **Redo** undo and redo the edits of the current list, including a change of its type.
* Click a column header to sort the items by it, and drag the handle after a header to set the width of the column (double click it to make it automatic again).
* The **Filter** switch hides the items with the filter values of the fields, such as those that are done.

## List types

**List settings** sets the type of the current list, and whether the items with items in them, such as categories, show only their first field. **Edit type** edits the fields of the type of the current list, and **List types** edits the library of types: drag the fields or types by their index to reorder them. A new list has the type of the current list, or the first type when no list is current. The standard types are Todo, Shopping, Project, Reading, and Simple.

Each field has:

* **Name**: the name of the column. Renaming a field keeps its values.
* **Type**: Text, Check (a check box), Integer, Number, Date, DateTime, or Duration (such as `1h30m`). A value of another type is dropped when the type changes.
* **Default**: the value of a new item, such as `1` for a quantity.
* **Tags**: more options for the values, such as `min:"1" max:"5"` or `step:"0.5"` for a number, `format:"%.2f"` for its format, and `edit:"-"` for no editing.
* **Set by**: for a date field, the name of a check field, such as Done, whose checking sets the date to today, as for Done on.
* **Filter**: a value, such as `true` for a check field like Done, for which the items are hidden when the list is filtered. Items with a filter value are also *done*, for reminders.
* **Flags**: **Detail** for a field that is only in the form of an item, which its edit button opens, as for notes; **Remind** for a date field whose dates have reminders, as for Due.

A list type also has a **Reminder** time (see below).

## Reminders

The items that are not done remind you of the dates of their fields with the **Remind** flag, such as Due in the Todo and Project types, with the notifications of your device. A notification shows the item and its list, and clicking it opens the list. The operating system shows them even when the app is not running, except on Linux and the web, where they are only shown while the app is running.

* A date without a time reminds you at the **Reminder** time of the list type, such as `9:00` or `5:30pm`, or, if it is empty, at the time in the **Reminders** settings (9:00 by default). A date and time reminds you at that time.
* Turn reminders off or on in the **Reminders** settings (in the app's Settings). Turning them on asks for permission to show notifications, if it is not given yet; you can also allow them in the settings of your system.
* On macOS, the app must be in `/Applications` or `~/Applications` for its notifications to be allowed.

## Syncing with GitHub

The lists can be synced with a GitHub repository: the app merges the changes in the repository with those on this device, and commits the merged lists, so that every device has the same lists, and the repository has the history of all of the changes. Syncing happens after edits (after a short delay, or right away on phones), at an interval, when you come back to the app, and when you quit it. The sync button in the toolbar shows the state of the last sync, and syncs now when clicked.

To set it up:

1. **Make a repository** for the lists on [GitHub](https://github.com/new), such as a new **private** one named `lists`. It can be empty; the files of the lists are put in its top directory.
2. **Make a token** that lets the app read and write the repository: go to [github.com/settings/personal-access-tokens](https://github.com/settings/personal-access-tokens) (Settings, Developer settings, Personal access tokens, **Fine-grained tokens**) and click **Generate new token**.
   * Give it a name, such as `Cogent Lists`, and an expiration (it must be renewed when it expires).
   * Under **Repository access**, choose **Only select repositories**, and select the repository of the lists.
   * Under **Permissions**, set **Contents** to **Read and write** (Metadata is then read-only, which is needed).
   * Click **Generate token**, and copy it: GitHub only shows it once.
3. **Enter the settings** in the app: open **Settings** in the app menu, and under **Sync**, set:
   * **Repo**: the repository, as `owner/name`, such as `yourname/lists`.
   * **Branch**: the branch of the lists, `main` by default.
   * **Token**: the token you copied.
   * **Interval**: how often to check for changes in the repository, in minutes.

Do the same on each of your devices, with the same repository (you can use the same token, or one for each device). The first sync of a new device with no lists of its own just loads the lists of the repository; otherwise, the lists of the device and of the repository are merged.

How the merge works: lists, groups, items, and list types are merged by their identity, so that renaming or moving them is kept. A change on one device is kept; when the same value is changed on two devices, the change that is synced last wins. Something that is deleted on one device is deleted, unless it was changed on the other. The repository keeps every version, so nothing is lost: you can look at the history of the files on GitHub.

If syncing fails, the sync button shows the error, such as a token that is not valid or has expired, or that cannot write the repository (it needs the Contents permission for Read and write).

## Files

The lists are saved in the app's data directory, as JSON files, which are those in the repository when syncing:

* `tree.json`: the tree of the lists and groups.
* `types.json`: the list types.
* `lists/`: a file for each list, with its items, named by the ID of the list.

The column widths of the lists, which depend on the size of the screen, are in `local.json`, and the state of the last sync is in `sync.json`; these are not synced. The desktop app opens the lists in another directory if it is given as its argument.

## Keyboard

* `Control+Tab` and `Control+Shift+Tab` move the focus between the tree of lists and the items (also `Control+X o` and `Control+X p` with the emacs key maps, and `Control+E o` and `Control+E p` with the standard Linux and Windows key maps).
* The undo and redo keys of your key map undo and redo the edits of the current list, and the insert keys add items after the selected one or in it, as shown in the tooltips of the toolbar buttons.
* `Command+]` and `Command+[` indent and outdent the selected items.
