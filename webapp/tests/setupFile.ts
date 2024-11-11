import React from 'react'
import { jest } from '@jest/globals'

jest.spyOn(global.PostUtils, 'messageHtmlToComponent').mockImplementation(() => {
    React.createElement('div', { className: 'mocked-message-html' }, 'Test Comment')
})
