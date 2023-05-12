import React from 'react'

import TextareaAutosize from 'react-textarea-autosize';

export default React.forwardRef((props, ref) => {
    const { onEnter, ...rest } = props;

    return (<TextareaAutosize
        {...rest}
        ref={ref}
        onKeyDown={event => onKeyDown(event, onEnter)}
    />);
}
);

function onKeyDown(event, f) {
    if (event.keyCode === 13) {
        f();
        event.preventDefault();
    }
}