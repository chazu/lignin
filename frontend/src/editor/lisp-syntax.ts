import { LanguageSupport, StreamLanguage, StreamParser } from '@codemirror/language';
import { tags as t } from '@lezer/highlight';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';

const KEYWORDS = new Set([
  'defpart', 'board', 'material', 'butt-joint', 'assembly',
  'part', 'place', 'def', 'list', 'vec3', 'screw', 'define',
]);

interface LispState {
  /** Depth of nested parentheses (not used for parsing, but available). */
  depth: number;
}

const lispStreamParser: StreamParser<LispState> = {
  startState(): LispState {
    return { depth: 0 };
  },

  token(stream, state): string | null {
    // Skip whitespace
    if (stream.eatSpace()) return null;

    // Comments: ; to end of line
    if (stream.match(/^;.*/)) {
      return 'comment';
    }

    // Strings
    if (stream.match(/^"(?:[^"\\]|\\.)*"/)) {
      return 'string';
    }
    // Unterminated string (consume rest of line)
    if (stream.match(/^"(?:[^"\\]|\\.)*$/)) {
      return 'string';
    }

    // Parentheses
    if (stream.eat('(')) {
      state.depth++;
      return 'bracket';
    }
    if (stream.eat(')')) {
      state.depth = Math.max(0, state.depth - 1);
      return 'bracket';
    }

    // Keyword arguments starting with :
    if (stream.match(/^:[a-zA-Z_\-][a-zA-Z0-9_\-]*/)) {
      return 'propertyName';
    }

    // Numbers (including negative, decimals)
    if (stream.match(/^-?\d+\.?\d*/)) {
      return 'number';
    }

    // Identifiers and keywords
    if (stream.match(/^[a-zA-Z_\-][a-zA-Z0-9_\-]*/)) {
      const word = stream.current();
      if (KEYWORDS.has(word)) {
        return 'keyword';
      }
      return 'variableName';
    }

    // Advance past any unrecognized character
    stream.next();
    return null;
  },
};

const lispStreamLanguage = StreamLanguage.define(lispStreamParser);

/**
 * Syntax highlighting colours tuned for the dark Catppuccin-ish theme.
 */
export const lispHighlight = syntaxHighlighting(
  HighlightStyle.define([
    { tag: t.keyword, color: '#cba6f7' },          // Mauve
    { tag: t.string, color: '#a6e3a1' },            // Green
    { tag: t.number, color: '#fab387' },             // Peach
    { tag: t.comment, color: '#6c7086', fontStyle: 'italic' }, // Overlay0
    { tag: t.propertyName, color: '#89b4fa' },       // Blue
    { tag: t.bracket, color: '#f9e2af' },            // Yellow
    { tag: t.variableName, color: '#cdd6f4' },       // Text
  ]),
);

/**
 * Full LanguageSupport instance for Lisp (Lignin dialect).
 */
export const lispLanguage = new LanguageSupport(lispStreamLanguage);
