import {useState} from 'react';
import './App.css';
import {Greet} from "../wailsjs/go/main/App";

import { Quit } from '../wailsjs/runtime/runtime'

function App() {
    const [resultText, setResultText] = useState("Please enter your name below 👇");
    const [name, setName] = useState('');
    const updateName = (e: any) => setName(e.target.value);

    function joinServer() {
        console.log('Joining server: ' + name);
        Greet(name);
    }

    return (
        <div id="app" className='bg-zinc-900 border-zinc-700 border'>
            <nav className="draggable flex px-2 justify-between items-center py-2 bg-zinc-800 shadow-lg">
                <div>
                    minepkg
                </div>
                <div className='flex gap-2'>
                    <div className='p-2 px-4 cursor-default rounded hover:bg-white/10' onClick={Quit}>
                        -
                    </div>
                    <div className='p-2 px-4 cursor-default rounded hover:bg-white/10' onClick={Quit}>
                        O
                    </div>
                    <div className='p-2 px-4 cursor-default rounded hover:bg-white/10' onClick={Quit}>
                        x
                    </div>
                </div>
            </nav>
            <div id="input" className="flex gap-2 justify-center p-8">
                <input id="name" className="p-2 rounded-lg shadow bg-zinc-800 text-white" placeholder="Server IP Address" onChange={updateName} autoComplete="off" name="input" type="text"/>
                <button className="p-2 px-4 bg-orange-600 shadow-orange-600/30 shadow-lg font-bold rounded-lg" onClick={joinServer}>Join Server</button>
            </div>
        </div>
    )
}

export default App
