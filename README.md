# DsPy

DSPy is a `super exciting` new framework for developing` LLM programs`! 

Pioneered by frameworks such as LangChain and LlamaIndex, we can build much more powerful systems by chaining together LLM calls! This means that the output of one call to an LLM is the input to the next, and so on. We can think of chains as programs, with each LLM call analogous to a function that takes text as input and produces text as output. Provides a general purpose modules which replaces with string-based prompting and also, helps in automating the Language Models.
## Programming, **NOT** Prompting

It is an LLM Programming language.
- Clean up your prompts and structured input/outputs
- Control how your LLM modules interact with each other programatically.
- Streamline the process of prompting with an input and objective.
- Create Modules of LMs that can work to achieve objective.
- Then train the modules using Datasets built by hand, and by using LLMs.
- Evaluate the modules with the Metrics that is custom coded in `DSPy`
- Including FineTuning smaller models like BerT, T5.

## Modules that DSPy has?
- Lang Models
- Signatures
- Modules
- Data
- Metrics
- Optimizers
- Assertions
## Use cases of DSPy?
QA, Classification, Summarisation, RAGS / Multi-Hop rags, Reasoning

---

## Instructions to Run:

Here’s how you can set up a Python virtual environment (`venv`) for your project, which will allow you to manage dependencies in isolation:

### Step 1: Install Python and `venv`
Make sure you have **Python 3** installed. You can check your Python version by running:

```bash
python3 --version
```

The `venv` module should come pre-installed with Python 3. If not, you can install it by running:

```bash
sudo apt-get install python3-venv  # For Ubuntu/Debian
```

### Step 2: Create a Virtual Environment
Navigate to your project directory and create a virtual environment. Run the following commands:

```bash
# Navigate to your project directory
cd /path/to/your/project

# Create the virtual environment (you can replace 'venv' with any other name)
python3 -m venv venv
```

### Step 3: Activate the Virtual Environment
Activate the virtual environment:

- **For Linux/macOS:**
  ```bash
  source venv/bin/activate
  ```

- **For Windows:**
  ```bash
  venv\Scripts\activate
  ```

Once activated, your terminal prompt will change to show the virtual environment name.

### Step 4: Install Required Packages
With the virtual environment activated, you can now install the necessary dependencies for your project. For your case, you may want to install `FastAPI`, `dspy`, and other dependencies:

```bash
pip install fastapi uvicorn dspy openai
```

### Step 5: Save Dependencies
After installing the required packages, it’s a good idea to save them in a `requirements.txt` file, which can be used to recreate the environment later:

```bash
pip install -r requirements.txt
```

### Step 6: Deactivate the Virtual Environment
When you are done working, you can deactivate the virtual environment by running:

```bash
deactivate
```

### Step 7: Reactivate the Virtual Environment (When Needed)
Whenever you come back to the project, simply reactivate the virtual environment:

- **For Linux/macOS:**
  ```bash
  source venv/bin/activate
  ```

- **For Windows:**
  ```bash
  venv\Scripts\activate
  ```

That’s it! You now have a fully isolated Python environment for your project.



