import { OpenFile, SaveFile, SetTitle } from '../wailsjs/go/main/App';

export interface FileState {
  path: string;       // empty string = untitled
  dirty: boolean;
  content: string;    // last saved content
}

export interface FileManager {
  open(): Promise<void>;      // Cmd-O
  save(): Promise<void>;      // Cmd-S
  markDirty(): void;          // called on editor change
  isDirty(): boolean;
  getState(): FileState;
}

/**
 * Creates a FileManager that coordinates file I/O state.
 *
 * @param getContent  - returns the current editor content
 * @param setContent  - replaces the editor content (used by open)
 * @param onStateChange - called whenever FileState changes (for UI updates)
 */
export function createFileManager(
  getContent: () => string,
  setContent: (content: string) => void,
  onStateChange: (state: FileState) => void
): FileManager {
  const state: FileState = {
    path: '',
    dirty: false,
    content: '',
  };

  function notify(): void {
    onStateChange({ ...state });
  }

  function updateTitle(): void {
    const filename = state.path
      ? state.path.split('/').pop()!.split('\\').pop()!
      : 'Untitled';
    const dirtyMark = state.dirty ? ' \u2022' : '';
    SetTitle(`Lignin \u2014 ${filename}${dirtyMark}`);
  }

  return {
    async open(): Promise<void> {
      const result = await OpenFile();
      // User cancelled or empty result.
      if (!result || !result.path) {
        return;
      }
      state.path = result.path;
      state.content = result.content;
      state.dirty = false;
      setContent(result.content);
      updateTitle();
      notify();
    },

    async save(): Promise<void> {
      const currentContent = getContent();
      const savedPath = await SaveFile(currentContent, state.path);
      // User cancelled the save dialog.
      if (!savedPath) {
        return;
      }
      state.path = savedPath;
      state.content = currentContent;
      state.dirty = false;
      updateTitle();
      notify();
    },

    markDirty(): void {
      if (!state.dirty) {
        state.dirty = true;
        updateTitle();
        notify();
      }
    },

    isDirty(): boolean {
      return state.dirty;
    },

    getState(): FileState {
      return { ...state };
    },
  };
}
