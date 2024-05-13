const wordSize = 4;

class Stack {
	#cursor = 0;
	#data;

	constructor(size) {
		const buffer = new ArrayBuffer(size * wordSize);
		this.#data = new Int32Array(buffer);
	}

	push(value) {
		if (this.#cursor+1 >= this.#data.length) {
			throw new Error(`push: stack overflow - cursor: ${this.#cursor}, stack size: ${this.#data.length}`);
		}

		this.#data[this.#cursor] = value;
		this.#cursor++;
	}


	pop() {
		if (this.#cursor <= 0) {
			throw new Error(`pop: stack underflow - cursor: ${this.#cursor}, stack size: ${this.#data.length}`);
		}

		this.#cursor--;
		return this.#data[this.#cursor];
	}

	peek() {
		if (this.#cursor <= 0) {
			throw new Error(`peek: stack underflow - cursor: ${this.#cursor}, stack size: ${this.#data.length}`);
		}

		return this.#data[this.#cursor-1];
	}
}

//class Input {
//	// TODO: Implement
//}

class DiatomVM {
	static stackSize = 30;
	static ioBufferSize = 4096;
	static memorySize = 8192;

	#programCounter = 0;
	#dataStack = new Stack(this.stackSize);
	#returnStack = new Stack(this.stackSize);
	//#inputBuffer = new Input();
	//input = some element;
	//output = some element;
	#memory = new Uint8Array(new ArrayBuffer(this.memorySize));

	load(path) {
		fetch(path).then(response => {
		if (!response.ok) {
			throw new Error(`HTTP error! Status: ${response.status}`);
		}

		const program = new Uint8Array(response.arrayBuffer());
		});
	}
	// TODO: Implement
}

/*
  How do I want to use that stuff?

  <script src="diatom.js" />
  <script>
  const vm = DiatomVM.load("my-script.dia");
  vm.execute();
  vm.reset();
  </script>

  Handling Javascript events would immediately require some sort of
  event-loop/async programming capabilities but this is out of scope
  for now.
*/


const s = new Stack(30);
s.push(11);
console.log(s.peek());
console.log(s.pop());
console.log(s);

const vm = new DiatomVM();
vm.load("my-script.dia");
