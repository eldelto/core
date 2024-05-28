// Instructions
const EXIT = 0;
const NOP = 1;
const RET = 2;
const CONST = 3;
const FETCH = 4;
const STORE = 5;
const ADD = 6;
const SUBTRACT = 7;
const MULTIPLY = 8;
const DIVIDE = 9;
const MODULO = 10;
const DUP = 11;
const DROP = 12;
const SWAP = 13;
const OVER = 14;
const CJMP = 15;
const CALL = 16;
const SCALL = 17;
const KEY = 18;
const EMIT = 19;
const EQUALS = 20;
const NOT = 21;
const AND = 22;
const OR = 23;
const LT = 24;
const GT = 25;
const RPOP = 26;
const RPUT = 27;
const RPEEK = 28;
const BFETCH = 29;
const BSTORE = 30;
const DUMP = 31;

const wordSize = 4;
const wordMax = 2147483647;
const wordMin = -2147483648;

const stackSize = 30;
const ioBufferSize = 4096;
const memorySize = 8192;

class Stack {
	#cursor = 0;

	constructor(size) {
		const buffer = new ArrayBuffer(size * wordSize);
		this.data = new Int32Array(buffer);
	}

	push(value) {
		if (this.#cursor + 1 >= this.data.length) {
			throw new Error(`push: stack overflow - cursor: ${this.#cursor}, stack size: ${this.data.length}`);
		}

		this.data[this.#cursor] = value;
		this.#cursor++;
	}

	pop() {
		if (this.#cursor <= 0) {
			throw new Error(`pop: stack underflow - cursor: ${this.#cursor}, stack size: ${this.data.length}`);
		}

		this.#cursor--;
		return this.data[this.#cursor];
	}

	peek() {
		if (this.#cursor <= 0) {
			throw new Error(`peek: stack underflow - cursor: ${this.#cursor}, stack size: ${this.data.length}`);
		}

		return this.data[this.#cursor - 1];
	}
}

function add(a, b) {
	const c = a + b;
	if (c > wordMax) {
		return wordMax
	} else if (c < wordMin) {
		return wordMin
	} else {
		return c
	}
}

function subtract(a, b) {
	const c = a - b;
	if (c > wordMax) {
		return wordMax
	} else if (c < wordMin) {
		return wordMin
	} else {
		return c
	}
}

function multiply(a, b) {
	const c = a * b;
	if (c > wordMax) {
		return wordMax
	} else if (c < wordMin) {
		return wordMin
	} else {
		return c
	}
}

function boolToWord(b) {
	if (b) {
		return -1;
	}
	return 0;
}

class Input {
	#cursor = 0;
	#buffer = new Uint8Array(new ArrayBuffer(0));
	#resolve = null;

	pushData(data) {
		const remaining = this.#buffer.slice(this.#cursor);

		this.#buffer = new Uint8Array(remaining.length + data.length);
		this.#buffer.set(remaining);
		this.#buffer.set(data, remaining.length);

		this.#cursor = 0;

		if (this.#resolve) {
			const c = this.#buffer[this.#cursor];
			this.#cursor++;
			this.#resolve(c);
			this.#resolve = null;
		}
	}

	nextChar() {
		if (this.#cursor < this.#buffer.length) {
			const c = this.#buffer[this.#cursor];
			this.#cursor++;

			return Promise.resolve(c);
		}

		const promise = new Promise((res, _rej) => {
			this.#resolve = res;
		});

		return promise;
	}
}

class DiatomVM {
	#programCounter = 0;
	dataStack = null;
	returnStack = null;
	#inputBuffer = null;
	#inputElement = null;
	#outputElement = null;
	#memory = null;

	constructor() {
		this.reset();
	}

	validateMemoryAccess(addr) {
		if (addr >= this.#memory.length) {
			throw new Error(`out of bound memory access: programCounter=${this.#programCounter} address=${addr}`);
		}
	}

	fetchByte(addr) {
		this.validateMemoryAccess(addr);
		return this.#memory[addr];
	}

	storeByte(addr, b) {
		this.validateMemoryAccess(addr);
		this.#memory[addr] = b;
	}

	wordToBytes(w) {
		const bytes = new Uint8Array(new ArrayBuffer(wordSize));

		for (let i = 0; i < wordSize; i++) {
			bytes[i] = (w >> (i * 8)) & 0xFF;
		}

		return bytes;
	}

	fetchWord(addr) {
		let w = 0;
		for (let i = 0; i < wordSize; i++) {
			const b = this.fetchByte(addr + i);
			const shift = (wordSize - (i + 1)) * 8;
			w = w | (b << shift);
		}

		return w;
	}

	storeWord(addr, w) {
		const bytes = this.wordToBytes(w);
		for (let i = 0; i < wordSize; i++) {
			this.storeByte(addr + (wordSize - (i + 1)), bytes[i])
		}
	}

	handleInput = e => {
		if (e.key == "Enter") {
			var enc = new TextEncoder();
			const data = enc.encode(e.target.value + " ");
			this.#inputBuffer.pushData(data);
			e.target.value = "";
		}
	}

	withInput(selectorOrElement) {
		this.#inputElement = selectorOrElement;
		if (typeof selectorOrElement === "string") {
			this.#inputElement = document.querySelector(selectorOrElement);
		}

		this.#inputElement.addEventListener("keyup", this.handleInput);

		// Hack to fetch the initial text content of the element.
		const event = new Event("keyup");
		event.key = "Enter";
		this.#inputElement.dispatchEvent(event);

		return this;
	}

	withOutput(selectorOrElement) {
		this.#outputElement = selectorOrElement;
		if (typeof selectorOrElement === "string") {
			this.#outputElement = document.querySelector(selectorOrElement);
		}
		return this;
	}

	load(program) {
		if (program.length > this.#memory.length) {
			throw new Error(`program length (${program.length} bytes) exceeds available memory (${this.#memory.length} bytes)`);
		}

		this.#memory.set(program);
		return this;
	}

	loadRemote(path) {
		return fetch(path).then(response => {
			if (!response.ok) {
				throw new Error(`HTTP error! Status: ${response.status}`);
			}
			return response.arrayBuffer();
		}).then(data => {
			const program = new Uint8Array(data);
			this.load(program);
		});
	}

	reset() {
		if (this.#inputElement) {
			this.#inputElement.removeEventListener("keyup", this.handleInput);
		}

		this.#programCounter = 0;
		this.dataStack = new Stack(stackSize);
		this.returnStack = new Stack(stackSize);
		this.#inputBuffer = new Input();
		this.#inputElement = null;
		this.#outputElement = null;
		this.#memory = new Uint8Array(new ArrayBuffer(memorySize));
	}

	async execute() {
		while (true) {
			const instruction = this.#memory[this.#programCounter];

			switch (instruction) {
			case EXIT: {
				console.log("VM exited normally");
				return;
			}
			case NOP: {
				break;
			}
			case RET: {
				const addr = this.returnStack.pop();
				this.#programCounter = addr;
				continue;
			}
			case CONST: {
				this.#programCounter++;
				const w = this.fetchWord(this.#programCounter);
				this.dataStack.push(w);

				this.#programCounter += wordSize;
				continue
			}
			case FETCH: {
				const addr = this.dataStack.pop();
				const w = this.fetchWord(addr);
				this.dataStack.push(w);
				break;
			}
			case STORE: {
				const addr = this.dataStack.pop();
				const value = this.dataStack.pop();
				this.storeWord(addr, value);
				break;
			}
			case ADD: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(add(b, a));
				break;
			}
			case SUBTRACT: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(subtract(b, a));
				break;
			}
			case MULTIPLY: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(multiply(b, a));
				break;
			}
			case DIVIDE: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(b / a);
				break;
			}
			case MODULO: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(b % a);
				break;
			}
			case DUP: {
				const a = this.dataStack.peek();
				this.dataStack.push(a);
				break;
			}
			case DROP: {
				this.dataStack.pop();
				break;
			}
			case SWAP: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(a);
				this.dataStack.push(b);
				break;
			}
			case OVER: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(b);
				this.dataStack.push(a);
				this.dataStack.push(b);
				break;
			}
			case CJMP: {
				this.#programCounter++;
				const conditional = this.dataStack.pop();
				if (conditional === -1) {
					this.#programCounter = this.fetchWord(this.#programCounter);
				} else {
					this.#programCounter += wordSize;
				}
				continue
			}
			case CALL: {
				this.#programCounter++;
				this.returnStack.push(this.#programCounter + wordSize);

				this.#programCounter = this.fetchWord(this.#programCounter);
				continue
			}
			case SCALL: {
				this.#programCounter++;
				this.returnStack.push(this.#programCounter);

				this.#programCounter = this.dataStack.pop();
				continue
			}
			case KEY:{
				const b = await this.#inputBuffer.nextChar();
				this.dataStack.push(b);
				break;
			}
			case EMIT: {
				const value = String.fromCharCode(this.dataStack.pop());
				if (this.#outputElement) {
					this.#outputElement.textContent += value;
				} else {
					console.log(value);
				}
				break;
			}
			case EQUALS: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(boolToWord(b == a));
				break;
			}
			case NOT: {
				const a = this.dataStack.pop();
				this.dataStack.push(~a);
				break;
			}
			case AND: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(b & a);
				break;
			}
			case OR: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(b | a);
				break;
			}
			case LT: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(boolToWord(b < a));
				break;
			}
			case GT: {
				const a = this.dataStack.pop();
				const b = this.dataStack.pop();
				this.dataStack.push(boolToWord(b > a));
				break;
			}
			case RPUT: {
				const a = this.dataStack.pop();
				this.returnStack.push(a);
				break;
			}
			case RPOP: {
				const a = this.returnStack.pop();
				this.dataStack.push(a);
				break;
			}
			case RPEEK: {
				const a = this.returnStack.peek();
				this.dataStack.push(a);
				break;
			}
			case BFETCH: {
				const addr = this.dataStack.pop();
				const b= this.fetchByte(addr);
				this.dataStack.push(b);
				break;
			}
			case BSTORE: {
				const addr = this.dataStack.pop();
				const value = this.dataStack.pop();
				this.storeByte(addr, value);
				break;
			}
			case DUMP: {
				// Basically a no-op as dumping is not supported.
				this.dataStack.pop();
				break;
			}
			default:
				throw new Error(`unknown instruction '${instruction}' at memory address '${this.#programCounter}' - terminating`);
			}

			this.#programCounter++;
		}
	}
}

class DiatomRepl extends HTMLElement {
	static observedAttributes = ["src"];
	#vm = new DiatomVM();

	constructor() {
		super();
	}

	attributeChangedCallback(name, oldValue, newValue) {
		// TODO: Do we need this?
		console.log(
			`Attribute ${name} has changed from ${oldValue} to ${newValue}.`,
		);
	}

	connectedCallback() {
		const shadow = this.attachShadow({ mode: "open" });

		const wrapper = document.createElement("div");
		wrapper.setAttribute("class", "diatom-prompt");

		const output = document.createElement("output");
		output.setAttribute("class", "diatom-output");
		wrapper.appendChild(output);

		const input = document.createElement("input");
		input.setAttribute("type", "text");
		input.setAttribute("class", "diatom-input");
		wrapper.appendChild(input);

		// TODO: On Enter we should insert the text value of the input
		// into the output element

		shadow.appendChild(wrapper);

		this.#vm.withInput(input);
		this.#vm.withOutput(output);
		this.#vm.loadRemote(this.getAttribute("src"))
			.then(_ => this.#vm.execute());
	}
}

customElements.define("diatom-repl", DiatomRepl);
