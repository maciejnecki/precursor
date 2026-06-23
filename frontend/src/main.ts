import './style.css'
import { mount } from 'svelte'
import App from './App.svelte'

// Mount the root component using the Svelte 5 mount API.
const app = mount(App, {
  target: document.getElementById('app') as HTMLElement
})

export default app
