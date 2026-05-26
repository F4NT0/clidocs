# Folder System Refactoring

## Description

This file is intended to guide the AI on how the folder system should be designed in the clidocs project.

## Workflow

A few details before presenting the workflow:

1. When opening the clidocs program, there are 3 panels: the folder panel, the snippets panel, and the preview panel.
2. In the folder panel, when starting the clidocs project, it begins by displaying all folders within the parent folder. If the user simply types `clidocs` in the terminal, they can choose to go to the main project (`clidocs_snippets/`) or select another folder to open.
3. The main project called `clidocs_snippets` will be used as an example in the steps below, but the behavior must be implemented regardless of which parent folder is opened.

### Basic Folder Workflow

1. When the project is opened, the folder panel shows all folders inside the opened parent folder (`clidocs_snippets`).
2. When a folder with no subfolders (e.g. `work/`) is selected and Enter is pressed, the folder panel navigates into that directory, showing it as `~/` along with the snippets inside it.
3. When a folder with subfolders (e.g. `work/examples/`) is selected and Enter is pressed, a modal called "Select the subfolder" should open, displaying the internal folders of the `work` directory. The user can navigate between them and press Enter to access a subfolder. Navigation through nested subfolders must also be supported.
4. When focused on the folder panel in any folder and the lowercase `d` key is pressed, the folder creation modal should appear.
5. When a folder is selected in any folder and the uppercase `D` key is pressed, the subfolder creation modal should appear.


## Visual Workflow Examples

### Navigating Between Folders

#### Without Subfolders

- Folder panel at the parent folder:

```txt
# Opening the parent folder shows the following example projects
# Note: folder icons should be present as already implemented; the list below represents folders

Work
Examples > 
Directories > 

# In the example above, the > symbol indicates the presence of subfolders
```

- After accessing the Work folder:

```txt
# When the ~/ entry is selected in this view,
# all text files will be shown in the snippet panel.
# The first directory in the folder panel should always be active
# so its snippets are visible in the snippet panel.

~/

```

#### With Subfolders

- Folder panel at the parent folder:

```txt

# Opening the parent folder shows the following example projects
# Note: folder icons should be present as already implemented; the list below represents folders

Work
Examples > 
Directories > 

# In the example above, the > symbol indicates the presence of subfolders
```

- Pressing Enter on the Examples directory:

1. A modal opens showing the internal directories.

```txt
# This is just an example of the modal:

Select the subfolder to open

> Examples/Studies/
Examples/Workflows/
Examples/Enable/

Enter Select | -> Access subfolders | <- Back to parent | q Exit

# > works the same as in the folder panel, for selection
# Pressing Enter on the selected subfolder opens it in the folder panel
# Pressing -> (right arrow) shows subfolders inside the current subfolder
# Pressing <- (left arrow) goes back one subfolder level, up to the parent folder where the project was opened

# Example after pressing ->:

Select the subfolder to open

> Examples/Studies/my-folder1/
Examples/Studies/my-folder2/

Enter Select | -> Access subfolders | <- Back to parent | q Exit

# Note: -> can be used to navigate all the way to the deepest subfolder
# Note 2: <- can be used to navigate back all the way to the parent folder where the program was opened

```

2. After the user makes a selection, the modal closes and the folder panel view updates.
3. The folder panel displays the name of the parent folder of the selected subfolder.
4. `~/` shows the snippets of the parent folder.
5. All directories of the current folder are displayed.


### Creating New Folders

1. In the folder panel, pressing lowercase `n` opens the folder creation modal.
2. In the folder panel, pressing uppercase `N` opens the subfolder creation modal for the currently selected folder.
3. If a subfolder is accessed, this logic must continue to work.

### Favoriting Folders

1. In the folder panel, pressing lowercase `d` favorites the selected folder.
2. In the folder panel, pressing uppercase `D` opens the favorites modal.
3. It must be possible to favorite any folder or subfolder.
4. The favorites modal must show the full path to the favorited folder, starting from the parent folder.

### Favorites Modal

1. The modal must allow selecting a favorite with Enter, as is standard in the system.
2. If both the parent folder and a subfolder have been favorited, both paths must appear in the list.
3. The full path to each folder must be shown, to distinguish between folders with the same name in different locations.
4. Pressing lowercase `o` in the modal opens the folder using the Windows file explorer.
5. Pressing `Enter` navigates to the favorite, updating the folder panel to show the selected folder.
6. In the folder panel, `~/` is shown as the selected folder with a star beside it to indicate it is a favorite.

### Location Switch Modal

1. When in the folder panel in any directory, the user should be able to press lowercase `o`.
2. Pressing lowercase `o` should open a modal showing the current location of the open parent directory (already exists).
3. This modal must include the option to change the location, i.e., the user can open the program in any other folder on the computer.
4. When a new folder is opened, it becomes the software's parent folder.

### Folder Deletion

1. Pressing lowercase `x` on the keyboard while a folder is selected opens the deletion modal.
2. The modal displays the folder name and asks for confirmation before deleting.
3. It must be possible to delete any folder in any parent folder or subfolder, always showing the full path to the folder.
4. Pressing uppercase `X` in the folder panel enables multi-folder selection for deletion.
5. When selecting multiple folders, the name of each folder to be deleted should be highlighted in red.
6. The `Space` key is used to select folders one by one.
7. Pressing `Enter` opens a modal listing all selected folders to confirm the deletion.

### Folder Rename

1. Pressing lowercase `r` on the keyboard while a folder is selected opens the rename modal.
2. A validation must be in place during renaming: the new name cannot contain spaces (e.g., `name subname/` is not allowed).
3. An error message should be shown if the user tries to save a name with spaces; the allowed formats are `user_name` or `user-name`.
4. If the renamed folder is in the favorites list, its name must be updated there as well.

### Folder Search

1. In the folder panel, pressing `/` on the keyboard should open a modal called "Search Folder".
2. There must be an input area where, as the user types, all folders matching the typed characters are shown below.
3. If there are folders named "Test" and "Testimony" and the user types "te", only directories beginning with those letters should be shown.
4. The search must also work inside subdirectories; for example, if `work/test` or `work/func/Test` exists, they should appear in the results.
5. This search must be activatable by pressing `/` from any folder or subfolder in the software.
6. When the desired folder is selected and `Enter` is pressed, the folder panel navigates to that folder.
