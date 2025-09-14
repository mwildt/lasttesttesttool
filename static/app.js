import {LitElement, css, html} from './lit.js';

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

function event(key, detail) {
    return new CustomEvent(key, {bubbles: true, composed: true, detail: detail});
}

function requestToastEvent(title, message, options) {
    return new CustomEvent("app-toaster::request-toast", {bubbles: true, composed: true, detail: {
        title: title,
        message: message,
        ...{type: "default", ...options}
    }});
}

customElements.define('system-info', class extends LitElement {

    static properties = {
        info: {}
    }

    constructor() {
        super();
        this.info = undefined
    }

    connectedCallback() {
        super.connectedCallback()
        fetch("/system-info")
            .then(r => r.json())
            .then(r => this.info = r)
    }

    render() {
        if (!this.info) {
            return html`asdasjd`
        } else {
            return html`<div>
                <span>${new Date(this.info.buildTimestamp).toUTCString()}</span> | <span>${this.info.buildBranch}</span> | <span>${this.info.commitId}</span>
            </div>`
        }

    }
})

customElements.define('app-toaster', class extends LitElement {

    static properties = {
       toasts: { type: Array }
    };

    static styles = css`
        .toaster {
          position: fixed;
          top: 0;
          right: 0;
          margin: 1rem; 
        }
        .default {
            color: #1e3a8a; 
            background-color: #dbeafe;
            border: 2px solid #93c5fd;
        }
        .warn {
            color: #78350f;
            background-color: #fef3c7;
            border: 2px solid #fcd34d;
        }
        .success {
            color: #065f46;
            background-color: #d1fae5;
            border: 2px solid #6ee7b7;
        }
        .error {
            color: #991b1b;
            background-color: #fee2e2;
            border: 2px solid #fca5a5;
        }
        .toast {
            padding: 1rem;
            
            border-radius: 1rem;
            box-shadow: 0 4px 10px rgba(0,0,0,0.1);
            width: 20rem;
        }
    `

    constructor() {
        super()
        this.toasts = []
        this.addEventListener("app-toaster::request-toast", event => this.toast(event.detail))
    }

    render() {
        return html`
            <div class="toaster">
                ${this.toasts.map(toast => this.renderToast(toast))}
            </div>
            <slot></slot>
        `
    }

    toast(toast) {

        this.toasts = this.toasts.concat(toast)

        setTimeout(() => {
            this.toasts = this.toasts.map(i => {
                if (i === toast) {
                    i.hidden = true;
                }
                return i
            })

            setTimeout(() => {
                this.toasts = this.toasts.filter(i => i !== toast);
            }, 500);

        }, 2500)
    }

    renderToast(toast) {
        return html`<div 
                style="opacity: ${!toast.hidden ? 1 : 0}; transition: opacity 0.5s;" 
                class="toast ${toast.type}">${toast.message}</div>`;
    }
})

customElements.define('logout-btn', class extends LitElement {

    render() {
        return html`<w-button @click=${() => this.logout()}>Logout</w-button>`
    }

    logout() {
        fetch("/logout")
            .then(res => {
                if (res.ok) {
                    this.dispatchEvent(event("logout-btn::logout", {}))
                } else {
                    this.dispatchEvent(requestToastEvent("Logout", "Logout fehlgeschlagen", {type: "error"}))
                }
            });
    }
})

customElements.define('metrics-container', class extends LitElement {

    static styles = css`
        .metrics {
            display: flex;
            flex-wrap: wrap;
            justify-content: space-between;
        }
        .metric {
            width: 20%
        }
    `

    static properties = {
        values: {}
    }

    constructor() {
        super();
        this.values = {}

        this.meta = {
            "bytes.write.count": {
                label: "Total Bytes written",
                formatter: formatBytes
            },
            "bytes.read.count": {
                label: "Total Bytes read",
                formatter: formatBytes
            },
            "request.count": {
                label: "Total Requests",
                formatter: a => a
            },
            "session.count": {
                label: "Active Sessions",
                formatter: a => a
            },
        }
    }

    connectedCallback() {
        super.connectedCallback()
        const eventSource = new EventSource('/stream', { withCredentials: true });
        eventSource.onmessage = () => {}
        eventSource.onerror = (err) => console.error(err);
        eventSource.addEventListener("ping", () => {});
        eventSource.addEventListener("store.event", (event) => {
            const decodedString = atob(event.data);
            const response = JSON.parse(decodedString);
            this.values = {...this.values, [response.key]:response.value}
        });
    }

    renderItem(key, description) {
        const value = this.values[key] !== undefined
            ? description.formatter(this.values[key])
            : "--";

        return html`<box-container class="metric">
            <h3>${description.label}</h3>
            <p>${value}</p>
        </box-container>`
    }

    render() {
        return html`<div>
            <h2>Metrics</h2>
            <div class="metrics">
                ${Object.entries(this.meta).map(([key, value]) => this.renderItem(key, value))}
            </div>
            <w-button @click=${e => this.reset()}>Reset</w-button>
        </div>`;
    }

    reset() {
        fetch("/reset", {method: "PATCH"})
            .then(() => this.dispatchEvent(requestToastEvent("Reset", "Reset erfolgreich", {type: "success"})))
            .catch(() => this.dispatchEvent(requestToastEvent("Reset", "Reset fehlgeschlagen", {type: "error"})))
    }
});

customElements.define('box-container', class extends LitElement {

    static styles = css`
    :host {
        display: block;
        margin-top: 1rem;
        font-weight: bold;
        color: #1e3a8a; 
        background-color: #dbeafe;
        padding: 2rem 3rem;
        border: 2px solid #93c5fd;
        border-radius: 1rem;
        box-shadow: 0 4px 10px rgba(0,0,0,0.1);
        text-align: center;
        min-width: 150px;
    }
    `

    render() {
        return html`<slot></slot>`
    }

})

customElements.define('w-button', class extends LitElement {

    static styles = css`
        button {
            margin-top: 1rem;
            padding: 0.8rem 2rem;
            font-size: 1.2rem;
            font-weight: bold;
            color: #1e3a8a; /* dunkles Blau */
            background-color: #dbeafe; /* hellblau */
            border: 2px solid #93c5fd; /* hellblauer Border */
            border-radius: 0.8rem;
            cursor: pointer;
            transition: all 0.2s ease;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        button:hover {
            background-color: #bfdbfe; 
            border-color: #60a5fa;
            box-shadow: 0 4px 10px rgba(0,0,0,0.15);
        }
        button:active {
            transform: translateY(1px);
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
    `

    _onClick(e) {
        this.dispatchEvent(new CustomEvent("button-click"));
    }

    render() {
        return html`<button @click=${this._onClick}><slot></slot></button>`
    }
})

customElements.define('app-container', class extends LitElement {

    static styles = css`
        :host {
            display: block;
            max-width: 800px;
            margin: 0px auto;
        }
  
        input {
            padding: 0.6rem;
            font-size: 1rem;
            border: 1px solid #60a5fa;
            border-radius: 0.5rem;
            width: 80%;
            margin-bottom: 1rem;
        }
    `

    static properties = {
        authenticated: {},
    }

    constructor() {
        super();
        this.authenticated = false
        this.key = undefined

        this.addEventListener("logout-btn::logout", () => {
                this.authenticated = false;
                this.dispatchEvent(requestToastEvent("Logout", "Logout erfolgreich", {type: "success"}));
        })
    }

    authenticate() {
        if (!this.key) {
            alert("Bitte Key eingeben");
            return;
        }
        fetch("/auth", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ key: this.key })
        }).then(res => {
            if (res.ok) {
                this.authenticated = true;
                this.key = undefined
                this.dispatchEvent(requestToastEvent("Login", "Login erfolgreich",{type: "success"}));
            } else {
                this.dispatchEvent(requestToastEvent("Login", "Login fehlgeschlagen", {type: "error"}));
            }
        });
    }

    renderLoginBox() {
        return html`<box-container>
                    <h2>Stop</h2>
                    <p>Gibt uns eine Chance dich zu kennen. Der Schlüssel liegt unter dem Stein.</p>
                    <input type="password" @change=${e => this.key = e.target.value} id="keyInput" placeholder="Enter key">
                    <br>
                    <w-button @click=${e => this.authenticate()}>Login</w-button>
                </box-container>`
    }

    render() {
        return html`
            <h1>Das Lasttesttesttool</h1>
            <p>This is where the Magic happens -- oder halt auch nicht... </p>
        
            ${this.authenticated 
                ? html`<logout-btn></logout-btn><metrics-container></metrics-container>`
                : this.renderLoginBox()}
                
            <p>By Malte Wildt, 48151 Münster -- Germany, thelasttesttesttool@maltewildt.de</p>
            <system-info></system-info>
        `
    }

});