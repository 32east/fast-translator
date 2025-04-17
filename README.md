Quick translator based on language models llama, chatgpt, and others (https://text.pollinations.ai/models).
## How to make this creation work?
1. Compile with the command ``go build -o build/fast-translator -ldflags \"-s -w\" -gcflags=all=\"-l -B\"``.
2. Run ``sudo build/fast-translator``. Administrator rights are needed to automatically install [XClip](https://github.com/astrand/xclip).
3. Select any text you want to translate.
4. Press the key combination: Ctrl + Shift + C.
## NOTICE!
Work on **Windows** is not checked (in theory, it should work, but with the condition that you need to copy the text first, and only after that press the combination Ctrl + Shift + C)!
<br>
Works on Unix-like systems only with the presence of the apt package manager, or if you manually installed [XClip](https://github.com/astrand/xclip).
