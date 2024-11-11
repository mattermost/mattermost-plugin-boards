import React from 'react'

(window as any).PostUtils = {
    messageHtmlToComponent: () => {
        React.createElement('div', { className: 'mocked-message-html' }, 'Test Comment')
    }
}
