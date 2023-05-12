import React from 'react';

import {
  HashRouter as Router,
  Switch,
  Route,
  useParams,
  useHistory
} from "react-router-dom";

import './App.css';
import DetailView from './solvent/render/DetailView'
import ListView from './solvent/render/ListView'

import { notebookFromDto, notebookToDto } from './solvent/Dto'

class App extends React.Component {

  constructor(props) {
    super(props);

    this.state = {
      notebook: null,
      isListViewActive: true
    }
  }

  componentDidMount() {
    this.syncState();
    this.timer = setInterval(() => this.syncState(), 1000);
  }

  componentWillUnmount() {
    clearInterval(this.timer);
    this.timer = null;
  }

  updateNotebook = f => {
    const notebook = this.state.notebook;
    const newNotebook = f(notebook);
    this.setState({ notebook: newNotebook });

    const activeToDoList = this.state.activeToDoList;
    if (activeToDoList) {
      const newActiveToDoList = notebook.getList(activeToDoList.id);
      if (newActiveToDoList) {
        const mergedActiveToDoList = activeToDoList.merge(newActiveToDoList);
        this.setState({ activeToDoList: mergedActiveToDoList });
      }
    }
  }

  syncState = async () => {
    if (this.state.notebook) {
      const newNotebook = await this.pushState(this.state.notebook);
      this.updateNotebook(notebook => notebook.merge(newNotebook));
    } else {
      const newNotebook = await this.fetchState();
      this.setState({ notebook: newNotebook });
    }
  }

  pushState = async notebook => {
    const dto = notebookToDto(notebook);
    const response = await fetch(process.env.REACT_APP_API_PATH + "/api/notebook", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(dto)
    });
    const responseBody = await response.json();
    return notebookFromDto(responseBody);
  }

  fetchState = async () => {
    // TODO: Fetch for real user
    const response = await fetch(process.env.REACT_APP_API_PATH + "/api/notebook/00000000-0000-0000-0000-000000000000");
    const responseBody = await response.json();
    return notebookFromDto(responseBody);
  }

  addList = (history) => {
    const list = this.state.notebook.addList("")
    this.setState({
      activeToDoList: list,
      notebook: this.state.notebook,
      isListViewActive: false
    });
    history.push("/list/" + list.id);
  }

  renameList = (list, title) => {
    this.updateNotebook(notebook => {
      list.rename(title);
      return notebook;
    });
  }

  checkItem = (list, item) => {
    if (item.checked) {
      this.updateNotebook(notebook => {
        list.uncheckItem(item.id);
        return notebook;
      });
    } else {
      this.updateNotebook(notebook => {
        list.checkItem(item.id);
        return notebook;
      });
    }
    document.activeElement.blur();
  }

  addItem = (list, title) => {
    this.updateNotebook(notebook => {
      list.addItem(title);
      return notebook;
    });
  }

  removeItem = (list, item) => {
    this.updateNotebook(notebook => {
      list.removeItem(item.id);
      return notebook;
    });

  }

  moveItem = (list, id, targetIndex) => {
    this.updateNotebook(notebook => {
      list.moveItem(id, targetIndex);
      return notebook;
    });
  }

  renameItem = (list, item, title) => {
    this.updateNotebook(notebook => {
      list.renameItem(item.id, title);
      return notebook;
    });
  }

  render() {
    return (
      <Router basename={process.env.PUBLIC_URL}>
        <div className={"App" + (this.state.isListViewActive ? " overview" : "")}>

          <Switch>
            <Route path="/list/:listId">
              <DetailViewContainer
                notebook={this.state.notebook}
                checkItem={this.checkItem}
                addItem={this.addItem}
                removeItem={this.removeItem}
                moveItem={this.moveItem}
                renameItem={this.renameItem}
                renameList={this.renameList}
              />
            </Route>

            <Route path="/">
              <ListViewContainer
                toDoLists={this.state.notebook ? this.state.notebook.getLists() : []}
                addList={this.addList}
              />
            </Route>
          </Switch>

        </div>
      </Router>
    );
  }
}

function ListViewContainer(props) {
  const history = useHistory();

  return (
    <div className="ViewContainer">
      <ListView
        {...props}
        addList={() => props.addList(history)}
      />
    </div>
  );
}

function DetailViewContainer(props) {
  const { listId } = useParams();
  if (listId === undefined || props.notebook === null) {
    return <h1>Not Found</h1>;
  }

  const toDoList = props.notebook.getList(listId);
  if (toDoList === undefined) {
    return <h1>Not Found</h1>;
  }

  return (<div className="ViewContainer">
    <DetailView
      {...props}
      toDoList={toDoList}
    />
  </div>);
}

export default App;
